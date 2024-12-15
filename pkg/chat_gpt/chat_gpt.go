package chat_gpt

import (
	"anki-tool/config"
	"anki-tool/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	apiPath     = "https://api.openai.com/v1/chat/completions"
	maxTokens   = 500
	temperature = 0.5
	modelName   = "gpt-4o-mini"

	prompt = `生成英文單字或片語的 json data
1.回傳的單純且有效的JSON陣列格式不需要code block包裝並遵守JSON規範, 支持多義詞, 將常見的不同詞性或不同含義作為獨立的對象返回
-vocabulary:單字
-part_of_speech:詞性（verb,noun,adj,adv）
-phonetic_transcription:音標
-definition:繁體中文字義
-forms: 列出不同於part_of_speech的詞性變化
-synonyms, antonyms: 
-phrases: 常搭配使用的介係詞或片語
-example_sentence, sentence_translation
2.{forms, synonyms, phrases} array length must between 0 to 2 
3.例句中使用 phrases 欄位中的短語, 單字片語 TOEIC 考試相關特別是職場或商務情境
範例：
input: approve
response: [{"vocabulary":"approve","part_of_speech":"verb","phonetic_transcription":"/əˈpruːv/","definition":"批准,同意","synonyms":["authorize (授權)","endorse (贊同;代言)"],"antonyms":["reject (拒絕)","disapprove (不贊成)"],"phrases":["approve a request (批准請求)"],"example_sentence":"The manager approved a request to extend the project deadline.","sentence_translation":"經理批准了延長專案期限的請求","forms":["approval(n)","approving(adj)","approvingly(adv)"]}]`
)

func NewOpenAIClient(cfg *config.Config) *OpenAIClient {
	return &OpenAIClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: cfg.OpenAIAPIKey,
	}
}

func (c *OpenAIClient) GenerateVocabularyInfo(ctx context.Context, word string) (result []model.VocabularyInfo, err error) {
	reqBody := RequestBody{
		Model: modelName,
		Messages: []Message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: word},
		},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("word: \"%s\" failed to marshal request body", word)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiPath, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("word: \"%s\" failed to create request", word)
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("word: \"%s\" API request failed", word)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("word: \"%s\" failed to read response body", word)
		return
	}

	var apiResp APIResponse
	if err = json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("word: \"%s\" failed to unmarshal response", word)
		return
	}

	if apiResp.Error != nil || len(apiResp.Choices) == 0 {
		log.Printf("word: \"%s\" response is null or error", word)
		return nil, fmt.Errorf("invalid API response")
	}

	log.Printf("word: \"%s\" Successfully call openAI API \n%v", word, apiResp.Choices[0].Message.Content)

	if err = json.Unmarshal([]byte(apiResp.Choices[0].Message.Content), &result); err != nil {
		log.Printf("word: \"%s\" failed to unmarshal response", word)
	}
	return
}
