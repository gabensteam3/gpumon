package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	botTokenEnv = "TELEGRAM_BOT_TOKEN"
	chatIDEnv   = "TELEGRAM_CHAT_ID"
)

func main() {
	// Get bot token and chat ID from environment variables
	botToken := os.Getenv(botTokenEnv)
	chatID := os.Getenv(chatIDEnv)

	if botToken == "" || chatID == "" {
		fmt.Printf("Environment variables %s and %s must be set.\n", botTokenEnv, chatIDEnv)
		return
	}

	// Read message from stdin
	message, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println("Error reading stdin:", err)
		return
	}

	// Prepare request
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	body := []byte(fmt.Sprintf("chat_id=%s&text=%s", chatID, string(message)))

	resp, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("Telegram API error: %s\n", respBody)
	} else {
		fmt.Println("Message sent successfully!")
	}
}

