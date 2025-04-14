package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const botTokenEnv = "TELEGRAM_BOT_TOKEN"

type Update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type GetUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

func main() {
	botToken := os.Getenv(botTokenEnv)
	if botToken == "" {
		fmt.Printf("Please set the environment variable %s with your bot token.\n", botTokenEnv)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error calling Telegram API:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var updates GetUpdatesResponse
	if err := json.Unmarshal(body, &updates); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	if !updates.Ok || len(updates.Result) == 0 {
		fmt.Println("No updates found. Make sure you've sent a message to your bot.")
		return
	}

	// Get the most recent chat ID
	chatID := updates.Result[len(updates.Result)-1].Message.Chat.ID
	fmt.Printf("Chat ID: %d\n", chatID)
}


