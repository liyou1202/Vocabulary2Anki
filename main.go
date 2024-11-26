package main

import (
	"anki-tool/config"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
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

type VocabularyInfo struct {
	Vocabulary            string   `json:"vocabulary"`
	PartOfSpeech          string   `json:"part_of_speech"`
	PhoneticTranscription string   `json:"phonetic_transcription"`
	Definition            string   `json:"definition"`
	Synonyms              []string `json:"synonyms"`
	Antonyms              []string `json:"antonyms"`
	Phrases               []string `json:"phrases"`
	ExampleSentence       string   `json:"example_sentence"`
	SentenceTranslation   string   `json:"sentence_translation"`
	Forms                 []string `json:"forms"`
}

const SheetName = "anki-en"
const CacheLimit = 50
const GoogleSheetCredentialPath = "./config/anki-en-credential.json"

// TODO: add Google sheet API to record gpt response
// TODO: if got "save" command get google sheet element and store as csv at local to batch insert to Anki
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// TODO: Using Webhook to minimize latency and sever burden
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 20

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	cache := NewCache(CacheLimit)
	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() && update.Message.Text == "" {
			continue
		}

		userInput := strings.ToLower(update.Message.Text)
		userInput = strings.TrimSpace(userInput)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Processing, wait a sec...")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending message: %v", err)
		}

		value, exists := cache.Get(userInput)
		response := value
		isNeedAppendToSheet := false
		if !exists {
			response, err = generateChatGPTResponse(userInput)
			if err != nil {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, err.Error())
				if _, err := bot.Send(msg); err != nil {
					log.Printf("Error sending message: %v", err)
				}
			}
			cache.Set(userInput, response)
			isNeedAppendToSheet = true
		}

		var info []VocabularyInfo
		err = json.Unmarshal([]byte(response), &info)
		if err != nil {
			log.Printf("Error parsing JSON: %v", err)
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "抱歉，處理資料時發生錯誤")
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending message: %v", err)
			}
			continue
		}

		for i, card := range info {
			formattedMessage := fmt.Sprintf(`
	第 %d 個
	📖 單字：*%s* (%s)
	🔤 音標：%s
	📝 定義：%s
	
	📝 詞性變化：
	%s
	👥 同義字：
	%s
	🔄 反義字：
	%s
	
	短語：
	%s
	📘 例句：_%s_
	⭐ 翻譯：%s`,
				i+1,
				card.Vocabulary,
				card.PartOfSpeech,
				card.PhoneticTranscription,
				card.Definition,
				formatStringArray(card.Forms),
				formatStringArray(card.Synonyms),
				formatStringArray(card.Antonyms),
				formatPhrases(card.Phrases),
				card.ExampleSentence,
				card.SentenceTranslation,
			)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, formattedMessage)
			msg.ParseMode = "Markdown" // 啟用 Markdown 格式

			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending message: %v", err)
			}

			// Append to google sheet
			if isNeedAppendToSheet {
				spreadsheetID := cfg.GoogleSheetId
				if err := AppendToGoogleSheet(spreadsheetID, info); err != nil {
					log.Fatalf("Error appending to Google Sheet: %v", err)
				}
			}
		}
	}

}

func formatStringArray(arr []string) string {
	if len(arr) == 0 {
		return "N/A"
	}
	return strings.Join(arr, ", ")
}
func formatPhrases(phrases []string) string {
	if len(phrases) == 0 {
		return "N/A"
	}
	result := ""
	for _, phrase := range phrases {
		result += "  - " + phrase + "\n"
	}
	return result
}

func generateChatGPTResponse(word string) (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", err
	}

	apiURL := "https://api.openai.com/v1/chat/completions"

	prePrompt := `生成英文單字或片語的 json data
1.format as json(keys：value explain)
-vocabulary: 單字
-part_of_speech: 詞性（verb, noun, adj, adv）
-phonetic_transcription: 音標
-definition: 繁體中文字義
-synonyms, antonyms: ()
-phrases: 常搭配使用的短語 or 介係詞  (array elements are empty if null or 1 value or 2 value) 
-example_sentence, sentence_translation
-forms: 如果存在不同詞性的變化列出 part_of_speech 以外的全部
1.每個陣列最多包含兩個元素。如果陣列沒有元素或只有一個元素，則返回空陣列或單一元素
2.回傳結果為單純的JSON不需要空格與code block，支持多義詞，將不同含義或不同詞性作為獨立的對象返回
3.例句中使用 phrases 欄位中的短語, 單字片語需與 TOEIC 考試相關特別是職場或商務情境
範例：[{"vocabulary":"approve","part_of_speech":"verb","phonetic_transcription":"/əˈpruːv/","definition":"批准,同意","synonyms":["authorize","endorse"],"antonyms":["reject","disapprove"],"phrases":["approve a request"],"example_sentence":"The manager approved a request to extend the project deadline.","sentence_translation":"經理批准了延長專案期限的請求。","forms":["approval(n)","approving(adj)","approvingly(adv)"]}]
`

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

// GetSheetHeader 獲取 Google Sheet 的標題列
func GetSheetHeader(service *sheets.Service, sheetID string) ([]string, error) {
	readRange := fmt.Sprintf("%s!1:1", SheetName) // 預設讀取第一列作為標題
	resp, err := service.Spreadsheets.Values.Get(sheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve sheet header: %v", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("no headers found")
	}

	headers := make([]string, len(resp.Values[0]))
	for i, header := range resp.Values[0] {
		headers[i] = fmt.Sprintf("%v", header)
	}

	return headers, nil
}

// AppendToGoogleSheet 將資料新增到 Google Sheet
func AppendToGoogleSheet(sheetID string, data []VocabularyInfo) error {
	ctx := context.Background()
	service, err := sheets.NewService(ctx, option.WithCredentialsFile(GoogleSheetCredentialPath))
	if err != nil {
		return fmt.Errorf("failed to create Sheets service: %v", err)
	}

	// 1. 獲取標題
	headers, err := GetSheetHeader(service, sheetID)
	if err != nil {
		return fmt.Errorf("unable to get headers: %v", err)
	}
	log.Printf("Headers: %v\n", headers)

	// 準備寫入的資料
	var valuesToWrite [][]interface{}

	for _, vocabItem := range data {
		rowData := make([]interface{}, len(headers))
		v := reflect.ValueOf(vocabItem)
		t := v.Type()

		for i, header := range headers {
			for j := 0; j < t.NumField(); j++ {
				field := t.Field(j)
				tag := field.Tag.Get("json")
				if tag == header {
					value := v.Field(j).Interface()
					switch val := value.(type) {
					case []string:
						rowData[i] = strings.Join(val, ", ")
					default:
						rowData[i] = val
					}
					break
				}
			}
		}
		valuesToWrite = append(valuesToWrite, rowData)
	}

	// 3. 獲取目前行數
	readRange := fmt.Sprintf("%s!A:A", SheetName) // 假設資料從 A 列開始
	resp, err := service.Spreadsheets.Values.Get(sheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("無法讀取行數: %v", err)
	}

	startRow := len(resp.Values) + 1

	// 4. 插入資料
	writeRange := fmt.Sprintf("%s!A%d", SheetName, startRow)
	writeBody := &sheets.ValueRange{
		Values: valuesToWrite,
	}

	_, err = service.Spreadsheets.Values.Update(sheetID, writeRange, writeBody).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("unable to write data to sheet: %v", err)
	}

	log.Printf("Successfully inserted %d rows to Google Sheet\n", len(valuesToWrite))
	return nil
}
