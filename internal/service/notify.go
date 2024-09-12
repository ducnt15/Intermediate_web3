package service

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
)

func SendMessage(message string) error {
	TelegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	groupIdStr := os.Getenv("TELEGRAM_CHAT_ID")
	bot, err := tgbotapi.NewBotAPI(TelegramBotToken)
	if err != nil {
		fmt.Println(err)
	}
	groupId, err := strconv.ParseInt(groupIdStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse GROUPCHAT_ID: %v", err)
	}
	chatId, err := strconv.Atoi(strconv.FormatInt(groupId, 10))
	if err != nil {
		fmt.Println(err)
	}
	// Create a new message to send
	msg := tgbotapi.NewMessage(int64(chatId), message)

	// Send the message
	_, err = bot.Send(msg)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
