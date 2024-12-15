package telegram

import (
	"anki-tool/config"
	"anki-tool/model"
	"anki-tool/pkg/cache"
	"anki-tool/pkg/chat_gpt"
	"anki-tool/pkg/google_sheet"
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"text/template"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

const cardTemplate = `
第 {{ .Index }} 個
📚 單字：*{{ .Vocabulary }}* ({{ .PartOfSpeech }})
🔤 音標：{{ .PhoneticTranscription }}
📝 定義：{{ .Definition }}
📝 詞性變化：{{ .Forms }}

🔗 同義字：{{ .Synonyms }}
↔️ 反義字：{{ .Antonyms }}
📌 短語：{{ .Phrases }}

💬 例句：
  {{ .ExampleSentence }}
  {{ .SentenceTranslation }}
`

type BotService struct {
	bot          *tgbotapi.BotAPI
	updates      tgbotapi.UpdatesChannel
	openAIClient *chat_gpt.OpenAIClient
	sheetClient  *google_sheet.SheetClient
	cache        *cache.Cache
}

func NewBotService(cfg *config.Config, openAIClient *chat_gpt.OpenAIClient, cache *cache.Cache, sheetClient *google_sheet.SheetClient) (*BotService, error) {
	newBot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, err
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 20
	updates, err := newBot.GetUpdatesChan(u)

	return &BotService{
		bot:          newBot,
		updates:      updates,
		openAIClient: openAIClient,
		sheetClient:  sheetClient,
		cache:        cache,
	}, nil
}

func (b *BotService) HandleUpdates() {
	log.Println("Start to monitor update from telegram chat")
	for update := range b.updates {
		if update.Message == nil || (!update.Message.IsCommand() && update.Message.Text == "") {
			continue
		}

		userInput := strings.ToLower(update.Message.Text)
		userInput = strings.TrimSpace(userInput)

		// send start query to telegram bot
		b.sendStrToChatRoom(update, "Processing, wait a sec...")

		flashcardWords, ok := b.cache.Get(userInput)
		if ok {
			b.sendCardToChatRoom(update, flashcardWords)
			continue
		}

		// call openAI API to gen vocabulary info
		flashcardWords, err := b.openAIClient.GenerateVocabularyInfo(context.Background(), userInput)
		if err != nil {
			failedStr := "failed to retrieve result from OpenAI API"
			log.Printf("word: \"%s\" %s \n%v", userInput, failedStr, err)
			b.sendStrToChatRoom(update, failedStr)
			continue
		}
		b.sendCardToChatRoom(update, flashcardWords)

		// append words to google sheet
		// success: append to cache
		// fail: send failed msg to telegram bot
		err = b.sheetClient.AppendToSheet(flashcardWords)
		if err != nil {
			failedStr := "failed to append data to google sheet"
			log.Printf("word: \"%s\" %s \n%v", userInput, failedStr, err)
			b.sendStrToChatRoom(update, failedStr)
			continue
		}

		err = b.cache.Set(userInput, flashcardWords)
		if err != nil {
			log.Printf("word: \"%s\" failed to set cache:\n%v", userInput, err)
		}
	}
}

func (b *BotService) sendStrToChatRoom(update tgbotapi.Update, str string) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, str)
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("failed to send message: %s", str)
	}
}

func (b *BotService) sendCardToChatRoom(update tgbotapi.Update, flashcardWords []model.VocabularyInfo) {
	for i, wordInfo := range flashcardWords {
		card, err := formatCardWithTemplate(i, wordInfo)
		if err != nil {
			log.Printf("failed to format card with template:\n%v", err)
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, card)
		msg.ParseMode = "Markdown"

		if _, err = b.bot.Send(msg); err != nil {
			log.Printf("failed to send message: %v", err)
		}
	}
}

func formatCardWithTemplate(index int, card model.VocabularyInfo) (string, error) {
	tmpl, err := template.New("card").Parse(cardTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Index":                 index + 1,
		"Vocabulary":            card.Vocabulary,
		"PartOfSpeech":          card.PartOfSpeech,
		"PhoneticTranscription": card.PhoneticTranscription,
		"Definition":            card.Definition,
		"Forms":                 formatPhrases(card.Forms),
		"Synonyms":              formatPhrases(card.Synonyms),
		"Antonyms":              formatPhrases(card.Antonyms),
		"Phrases":               formatPhrases(card.Phrases),
		"ExampleSentence":       card.ExampleSentence,
		"SentenceTranslation":   card.SentenceTranslation,
	})

	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

func formatPhrases(phrases []string) string {
	if len(phrases) == 0 {
		return "N/A"
	}
	result := ""
	for _, phrase := range phrases {
		result += "\n  " + phrase
	}
	return result
}
