package google_sheet

import (
	"anki-tool/config"
	"anki-tool/model"
	"anki-tool/pkg/cache"
	"context"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"log"
	"reflect"
	"strconv"
	"strings"
)

const GoogleSheetCredentialPath = "./config/anki-en-credential.json"

type SheetClient struct {
	service   *sheets.Service
	cache     *cache.Cache
	sheetID   string
	sheetName string
}

func NewSheetClient(cfg *config.Config, cache *cache.Cache) (*SheetClient, error) {
	ctx := context.Background()
	service, err := sheets.NewService(ctx, option.WithCredentialsFile(GoogleSheetCredentialPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create Sheets service: %w", err)
	}
	client := &SheetClient{
		service:   service,
		cache:     cache,
		sheetID:   cfg.GoogleSheetId,
		sheetName: cfg.GoogleSheetName,
	}

	all, err := client.fetchAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sheets all data: %w", err)
	}
	for key, value := range all {
		err := client.cache.Set(key, value)
		if err != nil {
			log.Printf("failed to append row to cache key: %s err: %v", key, err)
			return nil, err
		}
	}

	return client, nil
}

func (c *SheetClient) AppendToSheet(data []model.VocabularyInfo) error {
	//1. Retrieve sheet headers
	headers, err := c.getSheetHeaders()
	if err != nil {
		return fmt.Errorf("failed to get sheet headers: %w", err)
	}

	var valuesToWrite [][]interface{}
	for _, vocabItem := range data {
		rowData := c.mapVocabularyToRow(vocabItem, headers)
		valuesToWrite = append(valuesToWrite, rowData)
	}

	// 2. Get next available row
	readRange := fmt.Sprintf("%s!A:A", c.sheetName)
	resp, err := c.service.Spreadsheets.Values.
		Get(c.sheetID, readRange).
		Do()
	if err != nil {
		return fmt.Errorf("failed to determine starting row: %w", err)
	}

	startRow := len(resp.Values) + 1
	writeRange := fmt.Sprintf("%s!A%d", c.sheetName, startRow)

	writeBody := &sheets.ValueRange{
		Values: valuesToWrite,
	}

	// 3. Fill in data
	_, err = c.service.Spreadsheets.Values.Update(c.sheetID, writeRange, writeBody).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("failed to write data to sheet: %w", err)
	}

	log.Printf("Successfully inserted %d rows to Google Sheet", len(valuesToWrite))
	return nil
}

func (c *SheetClient) getSheetHeaders() ([]string, error) {
	readRange := fmt.Sprintf("%s!1:1", c.sheetName)
	resp, err := c.service.Spreadsheets.Values.Get(c.sheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve sheet headers: %w", err)
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
		return nil, fmt.Errorf("no headers found in sheet")
	}

	headers := make([]string, len(resp.Values[0]))
	for i, header := range resp.Values[0] {
		headers[i] = fmt.Sprintf("%v", header)
	}

	return headers, nil
}

func (c *SheetClient) mapVocabularyToRow(vocabItem model.VocabularyInfo, headers []string) []interface{} {
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
					// Join string arrays with comma
					rowData[i] = strings.Join(val, ", ")
				case string:
					// Trim whitespace for strings
					rowData[i] = strings.TrimSpace(val)
				default:
					rowData[i] = val
				}
				break
			}
		}
	}

	return rowData
}

func (c *SheetClient) fetchAll() (map[string][]model.VocabularyInfo, error) {
	header, err := c.getSheetHeaders()
	if err != nil {
		return nil, fmt.Errorf("failed to get sheet headers: %w", err)
	}

	lastColumn := string(rune('A' + len(header) - 1))
	readRange := fmt.Sprintf("%s!A:%s", c.sheetName, lastColumn)
	resp, err := c.service.Spreadsheets.Values.
		Get(c.sheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from sheet: %w", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("no data found in sheet")
	}

	dataMap := make(map[string][]model.VocabularyInfo, len(resp.Values))
	for i, row := range resp.Values[1:] {
		vocabulary, err := c.rowToVocabulary(row, header)
		if err != nil {
			log.Printf("failed to convert row to vocabulary at row %d, err: %v", i, err)
			continue
		}
		dataMap[vocabulary.Vocabulary] = append(dataMap[vocabulary.Vocabulary], vocabulary)
	}

	return dataMap, nil
}

func (c *SheetClient) rowToVocabulary(row []interface{}, headers []string) (model.VocabularyInfo, error) {
	vocabInfo := model.VocabularyInfo{}
	v := reflect.ValueOf(&vocabInfo).Elem()
	t := v.Type()

	for i, header := range headers {
		if i >= len(row) {
			continue
		}
		value := fmt.Sprintf("%v", row[i])
		for j := 0; j < t.NumField(); j++ {
			field := t.Field(j)
			tag := field.Tag.Get("json")
			if tag == header {
				fieldValue := v.Field(j)
				switch fieldValue.Kind() {
				case reflect.Slice:
					fieldValue.Set(reflect.ValueOf(strings.Split(value, ", ")))
				case reflect.String:
					fieldValue.SetString(value)
				case reflect.Int:
					intValue, err := strconv.Atoi(value)
					if err != nil {
						return vocabInfo, fmt.Errorf("failed to convert value to int: %w", err)
					}
					fieldValue.SetInt(int64(intValue))
				default:
					return vocabInfo, fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
				}
				break
			}
		}
	}
	return vocabInfo, nil
}
