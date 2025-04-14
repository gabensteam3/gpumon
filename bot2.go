package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

var telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
var apiUrl = "http://localhost:1101" // The URL of your backend API



func main() {
	bot, err := tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Create an update channel for receiving updates
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			// Handle callback queries (button clicks)
			handleCallback(update.CallbackQuery, bot)
		} else if update.Message != nil {
			// Handle messages
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handleStart(update.Message.Chat.ID, bot)
				case "gpus":
					handleGPUs(update.Message.Chat.ID, bot)
				case "hosts":
					handleHosts(update.Message.Chat.ID, bot)
				case "healthcheck":
					handleHealthCheck(update.Message.Chat.ID, bot)
				case "hardware":
					handleHardware(update.Message.Chat.ID, bot)
				default:
					handleUnknown(update.Message.Chat.ID, bot)
				}
			}
		}
	}
}

// /start command handler
func handleStart(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Hello 💖! I'm your server bot, here to help you manage your backend! 🌸✨")
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/gpus 🖥️", "gpus"),
			tgbotapi.NewInlineKeyboardButtonData("/hosts 📡", "hosts"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/hardware 🛠️", "hardware"),
			tgbotapi.NewInlineKeyboardButtonData("/health 🩺", "health"),
		),
	)

	msg.ReplyMarkup = inlineKeyboard
	bot.Send(msg)
}

// Callback query handler
func handleCallback(callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	// Get the callback data to identify which button was pressed
	data := callback.Data
	chatID := callback.Message.Chat.ID

	var response string
	switch data {
	case "gpus":
		response = "Fetching GPU information... 🖥️💖"
		handleGPUs(chatID, bot)
	case "hosts":
		response = "Fetching Host information... 📡💖"
		handleHosts(chatID, bot)
	case "hardware":
		response = "Fetching Hardware information... 🛠️💖"
		handleHardware(chatID, bot)
	case "health":
		response = "Checking system health... 🩺💖"
		handleHealthCheck(chatID, bot)
	default:
		response = "Unknown command. Please use the buttons again 💔"
	}

	// Send a response to acknowledge the callback
	callbackAnswer := tgbotapi.NewCallback(callback.ID, response)
	bot.AnswerCallbackQuery(callbackAnswer) // This line was updated to AnswerCallbackQuery instead of bot.Send
}






// /gpus command handler
func handleGPUs(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Fetching GPUs 💖✨")
	bot.Send(msg)

	// Get GPUs data from backend API
	resp, err := http.Get(fmt.Sprintf("%s/gpu/list", apiUrl))
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Sorry, I couldn't fetch the GPU data 😔💔")
		bot.Send(msg)
		return
	}
	defer resp.Body.Close()

	var gpus []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&gpus); err != nil {
		msg := tgbotapi.NewMessage(chatID, "Oops! Something went wrong while processing the data 😕💦")
		bot.Send(msg)
		return
	}

	if len(gpus) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No GPUs found 😢💔")
		bot.Send(msg)
		return
	}

	// Split GPU details into separate messages
	for _, gpu := range gpus {
		response := fmt.Sprintf("💎 *%s*\n", gpu["name"])
		response += fmt.Sprintf("Temperature: %d°C 🌡️\n", int(gpu["temperature_c"].(float64)))
		response += fmt.Sprintf("Fan Speed: %d%% 🌀\n", int(gpu["fan_percent"].(float64)))
		response += fmt.Sprintf("Power Usage: %.2f W ⚡\n", gpu["power_watt"])
		response += fmt.Sprintf("Memory Usage: %d MiB/%d MiB 💾\n", int(gpu["memory_used_mib"].(float64)), int(gpu["memory_total_mib"].(float64)))
		response += fmt.Sprintf("GPU Usage: %d%% 💪\n", int(gpu["utilization_gpu_percent"].(float64)))
		response += fmt.Sprintf("Processes: %d 👾\n", int(gpu["process_count"].(float64)))
		response += fmt.Sprintf("Updated At: %s ⏳\n", gpu["updated_at"])

		msg := tgbotapi.NewMessage(chatID, response)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	}
}

// /hosts command handler
func handleHosts(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Fetching Hosts 🖥️💖")
	bot.Send(msg)

	// Get Host data from backend API
	resp, err := http.Get(fmt.Sprintf("%s/host/list", apiUrl))
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Sorry, I couldn't fetch the host data 😔💔")
		bot.Send(msg)
		return
	}
	defer resp.Body.Close()

	var hosts []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&hosts); err != nil {
		msg := tgbotapi.NewMessage(chatID, "Oops! Something went wrong while processing the data 😕💦")
		bot.Send(msg)
		return
	}

	if len(hosts) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No hosts found 😢💔")
		bot.Send(msg)
		return
	}

	// Split host details into separate messages
	for _, host := range hosts {
		response := fmt.Sprintf("🌟 *%s*\n", host["hostname"])
		response += fmt.Sprintf("CPU Usage: %.1f%% 🧠\n", host["cpu_usage_percent"].(float64))
		response += fmt.Sprintf("Memory Usage: %d MB/%d MB 🧑‍💻\n", int(host["memory_used_mb"].(float64)), int(host["memory_total_mb"].(float64)))
		response += fmt.Sprintf("Disk Usage: %s/%s 🧳\n", host["disk_used"], host["disk_total"])
		response += fmt.Sprintf("Last Updated: %s ⏳\n", host["updated_at"])

		msg := tgbotapi.NewMessage(chatID, response)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	}
}

// /hardware command handler
func handleHardware(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Fetching Hardware Reports 🛠️💖")
	bot.Send(msg)

	// Get Hardware data from backend API
	resp, err := http.Get(fmt.Sprintf("%s/hardware/list", apiUrl))
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Sorry, I couldn't fetch the hardware data 😔💔")
		bot.Send(msg)
		return
	}
	defer resp.Body.Close()

	var hardwareReports []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&hardwareReports); err != nil {
		msg := tgbotapi.NewMessage(chatID, "Oops! Something went wrong while processing the data 😕💦")
		bot.Send(msg)
		return
	}

	if len(hardwareReports) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No hardware reports found 😢💔")
		bot.Send(msg)
		return
	}

	// Split hardware details into separate messages
	for _, report := range hardwareReports {
		response := fmt.Sprintf("🔧 *%s*\n", report["hostname"])
		response += fmt.Sprintf("Uptime: %s ⏳\n", report["uptime"])
		response += fmt.Sprintf("Kernel: %s 🐧\n", report["kernel"])

		msg := tgbotapi.NewMessage(chatID, response)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	}
}

func handleHealthCheck(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Checking system health... 🌸✨")
	bot.Send(msg)

	// Get health check data from backend API
	resp, err := http.Get(fmt.Sprintf("%s/healthcheck", apiUrl))
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Sorry, I couldn't reach the health check endpoint 😔💔")
		bot.Send(msg)
		return
	}
	defer resp.Body.Close()

	// Parse health check response
	if resp.StatusCode == http.StatusOK {
		msg := tgbotapi.NewMessage(chatID, "System health is OK ✅💖")
		bot.Send(msg)
	} else {
		// If the health check failed, we send the issues back to the user.
		var healthStatus map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthStatus); err != nil {
			msg := tgbotapi.NewMessage(chatID, "Oops! Something went wrong while processing the health check data 😕💦")
			bot.Send(msg)
			return
		}

		issues := healthStatus["issues"].([]interface{})
		var issueMessages []string
		for _, issue := range issues {
			issueMessages = append(issueMessages, issue.(string))
		}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("System health is not OK ❌💔\nIssues:\n- %s", fmt.Sprintf("%s", issueMessages)))
		bot.Send(msg)
	}
}

// /unknown command handler
func handleUnknown(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "Sorry 💕 I don't understand that command 😕💔\nPlease use /start to see available options.")
	bot.Send(msg)
}


