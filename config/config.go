package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	TelegramBotToken string `json:"telegram_bot_token"`
	OpenAIAPIKey     string `json:"openai_api_key"`
	GoogleSheetId    string `json:"google_sheet_id"`
}

var configuration *Config

func LoadConfig() (*Config, error) {
	if configuration != nil {
		return configuration, nil
	}

	file, err := os.ReadFile("config/config.json")
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	configuration = &Config{}
	err = json.Unmarshal(file, configuration)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	if configuration.TelegramBotToken == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	if configuration.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	return configuration, nil
}
