package config

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

type ChainConfig struct {
	Chain         string  `json:"chain"`
	Gpc           string  `json:"gpc"`
	UsersTracking string  `json:"usersTracking"`
	Tokens        []Token `json:"tokens"`
}

type Token struct {
	Symbol          string `json:"symbol"`
	ContractAddress string `json:"contract_address"`
}

var (
	Config           ChainConfig
	TelegramBotToken string
	TelegramChatID   string
)

func InitConfig() error {
	// Load environment variables from .env file
	err := loadEnv()
	if err != nil {
		return err
	}

	// Load configuration from config.json
	err = loadConfig()
	if err != nil {
		return err
	}

	TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	TelegramChatID = os.Getenv("TELEGRAM_CHAT_ID")

	if TelegramBotToken == "" || TelegramChatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is not set in the environment")
	}

	return nil
}

func loadEnv() error {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
		return err
	}
	return nil
}

func loadConfig() error {
	file, err := os.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("could not read config.json file: %w", err)
	}

	err = json.Unmarshal(file, &Config)
	if err != nil {
		return fmt.Errorf("could not parse config.json: %w", err)
	}
	return nil
}
