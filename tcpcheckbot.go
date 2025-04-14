package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// A map for tracking last command times for rate limiting
var userCommandTimes = make(map[int64]time.Time)

type Monitor struct {
	ID     int
	UserID int64
	Target string
	Up     bool
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	var err error
	db, err = sql.Open("sqlite3", "./monitors.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable()

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	go monitorServices(bot)

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		userID := update.Message.Chat.ID
		text := update.Message.Text

		// Rate limiting (1 command per minute per user)
		if time.Since(userCommandTimes[userID]) < time.Second {
			send(bot, userID, "⚠️ Please wait before issuing another command.")
			continue
		}
		userCommandTimes[userID] = time.Now()

		// Only allow authorized users (change user IDs to your own)
		allowedUserIDs := map[int64]bool{
			123456789: true, // Replace with your allowed user IDs
		}

		if allowedUserIDs[userID] {
			send(bot, userID, "❌ You are not authorized to use this bot.")
			continue
		}

		// Start command
		if text == "/start" {
			msg := `🤖 *Welcome to MonitorBot!*
This bot allows you to monitor the status of IP:Port addresses. You can add, delete, and list your monitors.

Commands:
- /add ip:port – ➕ Add a monitor
- /delete ip:port – 🗑️ Delete monitor
- /list – 📋 Show all monitors
- /help – ❓ Get help

Start managing your monitors by using the /add command!`

			message := tgbotapi.NewMessage(userID, msg)
			message.ParseMode = "Markdown"
			bot.Send(message)
			continue
		}

		// Help command
		if text == "/help" {
			msg := `🤖 *MonitorBot Help*
You can manage IP:Port monitors here.

Commands:
/add ip:port – ➕ Add a monitor
/delete ip:port – 🗑️ Delete monitor
/list – 📋 Show all monitors`

			message := tgbotapi.NewMessage(userID, msg)
			message.ParseMode = "Markdown"
			bot.Send(message)
		}

		if strings.HasPrefix(text, "/add") {
			args := strings.Fields(text)
			if len(args) != 2 {
				send(bot, userID, "⚠️ Usage: /add ip:port")
				continue
			}
			target := args[1]
			if !isValidIPPort(target) {
				send(bot, userID, "❌ Invalid IP:Port format.")
				continue
			}
			if err := addMonitor(userID, target); err != nil {
				send(bot, userID, "❌ Failed to add monitor or monitor already exists.")
			} else {
				send(bot, userID, "✅ Monitor added! 📡")
			}
		}

		if strings.HasPrefix(text, "/delete") {
			args := strings.Fields(text)
			if len(args) != 2 {
				send(bot, userID, "⚠️ Usage: /delete ip:port")
				continue
			}
			target := args[1]
			if !isValidIPPort(target) {
				send(bot, userID, "❌ Invalid IP:Port format.")
				continue
			}
			if err := deleteMonitor(userID, target); err != nil {
				send(bot, userID, "❌ Monitor not found or failed to delete.")
			} else {
				send(bot, userID, "🗑️ Monitor deleted!")
			}
		}

		if strings.HasPrefix(text, "/list") {
			rows, err := db.Query("SELECT id, target, up FROM monitors WHERE user_id = ?", userID)
			if err != nil {
				send(bot, userID, "❌ Could not list monitors")
				continue
			}
			defer rows.Close()

			msg := "📋 *Your Monitors:*\n"
			for rows.Next() {
				var id int
				var target string
				var up bool
				rows.Scan(&id, &target, &up)

				// Display the status
				status := "🟢 *UP*"
				if !up {
					status = "🔴 *DOWN*"
				}

				msg += fmt.Sprintf("%s - %s\n", status, target)
			}
			message := tgbotapi.NewMessage(userID, msg)
			message.ParseMode = "Markdown"
			bot.Send(message)
		}

		if text == "/start" || text == "/help" {
			msg := `🤖 *Welcome to MonitorBot!*
You can manage IP:Port monitors here.

Commands:
/add ip:port – ➕ Add a monitor
/delete ip:port – 🗑️ Delete monitor
/list – 📋 Show all monitors`
			message := tgbotapi.NewMessage(userID, msg)
			message.ParseMode = "Markdown"
			bot.Send(message)
		}
	}
}

func send(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	bot.Send(msg)
}

func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS monitors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		target TEXT UNIQUE,
		up BOOLEAN DEFAULT 1
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func addMonitor(userID int64, target string) error {
	// Check if the user already has a monitor for the target
	var existingTarget string
	err := db.QueryRow("SELECT target FROM monitors WHERE user_id = ? AND target = ?", userID, target).Scan(&existingTarget)
	if err == nil {
		// Monitor already exists for this user
		return fmt.Errorf("Monitor already exists")
	}

	_, err = db.Exec("INSERT INTO monitors (user_id, target) VALUES (?, ?)", userID, target)
	return err
}

func deleteMonitor(userID int64, target string) error {
	// Check if the monitor exists and belongs to the user
	result, err := db.Exec("DELETE FROM monitors WHERE user_id = ? AND target = ?", userID, target)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		// No rows affected means the monitor doesn't exist or it wasn't owned by the user
		return fmt.Errorf("Monitor not found or not owned by you")
	}
	return nil
}

func isValidIPPort(target string) bool {
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return false
	}
	if net.ParseIP(host) == nil {
		return false
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return false
	}
	return true
}

func monitorServices(bot *tgbotapi.BotAPI) {
	for {
		rows, err := db.Query("SELECT id, user_id, target, up FROM monitors")
		if err != nil {
			log.Println("Query error:", err)
			time.Sleep(30 * time.Second)
			continue
		}

		var monitors []Monitor
		for rows.Next() {
			var m Monitor
			rows.Scan(&m.ID, &m.UserID, &m.Target, &m.Up)
			monitors = append(monitors, m)
		}
		rows.Close()

		for _, m := range monitors {
			host, port, err := net.SplitHostPort(m.Target)
			if err != nil {
				continue
			}
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 5*time.Second)
			isUp := err == nil
			if conn != nil {
				conn.Close()
			}

			if isUp != m.Up {
				_, err := db.Exec("UPDATE monitors SET up = ? WHERE id = ?", isUp, m.ID)
				if err != nil {
					continue
				}
				status := "🔴 *DOWN*"
				if isUp {
					status = "🟢 *UP*"
				}
				msg := fmt.Sprintf("🔔 %s is now %s", m.Target, status)
				message := tgbotapi.NewMessage(m.UserID, msg)
				message.ParseMode = "Markdown"
				bot.Send(message)
			}
		}
		time.Sleep(30 * time.Second)
	}
}


