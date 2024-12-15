package main

import (
	"anki-tool/config"
	"anki-tool/pkg/cache"
	"anki-tool/pkg/chat_gpt"
	"anki-tool/pkg/google_sheet"
	"anki-tool/pkg/telegram"
	"io"
	"log"
	"os"
)

// TODO get google sheet data to cache
func main() {
	logFile, err := os.OpenFile("./output/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create the OpenAI client
	openAIClient := chat_gpt.NewOpenAIClient(cfg)

	// Create new cache
	newCache := cache.NewCache()

	// Create new sheet client
	sheetClient, err := google_sheet.NewSheetClient(cfg, newCache)
	if err != nil {
		log.Fatalf("Failed to initializing sheet client: %v", err)
	}

	// Initialize the Telegram Bot Service
	botService, err := telegram.NewBotService(cfg, openAIClient, newCache, sheetClient)
	if err != nil {
		log.Fatalf("Failed to initializing bot service: %v", err)
	}

	// Start handling updates
	botService.HandleUpdates()
}
