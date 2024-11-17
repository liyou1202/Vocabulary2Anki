package main

import (
	"anki-tool/config"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type ChatGPTRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	MaxTokens int `json:"max_tokens"`
}

type ChatGPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type VocabCard struct {
	Vocabulary            string   `json:"vocabulary"`
	PartOfSpeech          string   `json:"part_of_speech"`
	PhoneticTranscription string   `json:"phonetic_transcription"`
	Definition            string   `json:"definition"`
	Synonyms              []string `json:"synonyms"`
	Antonyms              []string `json:"antonyms"`
	RelatedWords          []string `json:"related_words"`
	ExampleSentence       string   `json:"example_sentence"`
	SentenceTranslation   string   `json:"sentence_translation"`
	Prepositions          []string `json:"prepositions"`
}

func main() {
	// 載入配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	cache := NewCache(50)
	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() && update.Message.Text == "" {
			continue
		}

		userInput := update.Message.Text

		value, exists := cache.Get(userInput)
		response := value
		if !exists {
			response, err = generateChatGPTResponse(userInput)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, err.Error())
				bot.Send(msg)
			}
			cache.Set(userInput, response)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Processing, wait a sec...")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending message: %v", err)
		}

		var vocabCard []VocabCard
		err = json.Unmarshal([]byte(response), &vocabCard)
		if err != nil {
			log.Printf("Error parsing JSON: %v", err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "抱歉，處理資料時發生錯誤")
			bot.Send(msg)
			continue
		}

		for i, card := range vocabCard {
			formattedMessage := fmt.Sprintf(`
第 %d 個
📖 單字：*%s* (%s)
🔤 音標：%s
📝 定義：%s

常搭配介系詞: %s
👥 同義字：%s
🔄 反義字：%s
🔍 相關字：%s

📘 例句：_%s_
⭐ 翻譯：%s`,
				i+1,
				card.Vocabulary,
				card.PartOfSpeech,
				card.PhoneticTranscription,
				card.Definition,
				formatStringArray(card.Prepositions),
				formatStringArray(card.Synonyms),
				formatStringArray(card.Antonyms),
				formatStringArray(card.RelatedWords),
				card.ExampleSentence,
				card.SentenceTranslation,
			)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, formattedMessage)
			msg.ParseMode = "Markdown" // 啟用 Markdown 格式

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending message: %v", err)
			}
		}
	}
}

func formatStringArray(arr []string) string {
	if len(arr) == 0 {
		return "N/A"
	}
	return strings.Join(arr, ",")
}

func generateChatGPTResponse(word string) (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}

	apiURL := "https://api.openai.com/v1/chat/completions"

	prePrompt := "輸入單字翻譯為繁體中文，生成Anki卡片資料：1.單字、詞性、音標、解釋（繁中）2.常搭配介係詞、同義字、反義字、相關字（最多2個可為空)3.例句(有搭配詞則要包含搭配詞)、例句翻譯（繁中)4.如果有多個意思則回傳陣列5.同義,反義,相關,例句以多使用多益單字為主6.若無\"prepositions\"則回傳\"\"7.僅回傳JSON，範例：[{\"vocabulary\":\"concern\",\"part_of_speech\":\"verb\",\"phonetic_transcription\":\"/kənˈsɜːrn/\",\"definition\":\"擔心,關心\",\"synonyms\":[\"worry\",\"care about\"],\"antonyms\":[\"disregard\",\"ignore\"],\"related_words\":[\"anxiety\",\"interest\"],\"example_sentence\":\"She was deeply concerned about the welfare of the local community.\",\"sentence_translation\":\"她非常關心當地社區的福祉。\",\"prepositions\":[\"about\"]},{\"vocabulary\":\"concern\",\"part_of_speech\":\"verb\",\"phonetic_transcription\":\"/kənˈsɜːrn/\",\"definition\":\"涉及,關係到\",\"synonyms\":[\"involve\",\"pertain to\"],\"antonyms\":[\"exclude\",\"irrelevant\"],\"related_words\":[\"connection\",\"significance\"],\"example_sentence\":\"This new policy concerns all employees in the company.\",\"sentence_translation\":\"這項新政策涉及公司所有員工。\",\"prepositions\":[\"\"]}]"

	requestBody := ChatGPTRequest{
		Model: "gpt-4o-mini",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: prePrompt},
			{Role: "user", Content: word},
		},
		MaxTokens: 500,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error marshaling request body: %v", err)
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling ChatGPT API: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return "", err
	}

	var chatGPTResp ChatGPTResponse
	if err := json.Unmarshal(respBody, &chatGPTResp); err != nil {
		log.Printf("Error unmarshaling response body: %v", err)
		return "", err
	}

	if len(chatGPTResp.Choices) > 0 {
		return chatGPTResp.Choices[0].Message.Content, nil
	}
	return "", errors.New("error parsing chatGPT resp")
}
