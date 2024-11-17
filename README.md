// README.md
# Vocabulary Learning Telegram Bot

A Telegram bot that helps users learn vocabulary with ChatGPT integration and record on Google sheet to generate Anki cards at the same time.

## Setup

1. Clone the repository
```bash
git clone [your-repo-url]
```

2. Copy the example config file and edit it with your tokens
```bash
cp config/config.example.json config/config.json
```

3. Edit `config/config.json` with your actual tokens:
- Add your Telegram Bot Token (from @BotFather)
- Add your OpenAI API Key

4. Build and run
```bash
go build
./[your-project-name]
```

## Configuration

The bot requires two API tokens:
- Telegram Bot Token: Obtained from Telegram's @BotFather
- OpenAI API Key: Obtained from OpenAI's website

Copy `config.example.json` to `config.json` and fill in your actual tokens.

## Security Notes

- Never commit your `config.json` file
- The `config.json` file is included in `.gitignore`
- Only commit `config.example.json` with placeholder values

## Prompt
```go
prePrompt := `
輸入單字翻譯為繁體中文，生成 Anki 卡片資料：
1. 單字、詞性、音標、解釋（繁中）
2. 同義字、反義字、相關字（最多2個可為空)
3. 例句、例句翻譯（繁中)
4. 如果有多個意思則回傳陣列
5. 同義, 反義, 相關, 例句以多使用多益單字為主
6. 如果有常搭配的介係詞，請新增欄位 "prepositions"，若無則回傳 ""
僅回傳 JSON，範例：
[{
  "vocabulary": "concern",
  "part_of_speech": "verb",
  "phonetic_transcription": "/kənˈsɜːrn/",
  "definition": "擔心, 關心",
  "synonyms": ["worry", "care about"],
  "antonyms": ["disregard", "ignore"],
  "related_words": ["anxiety", "interest"],
  "example_sentence": "She was deeply concerned about the welfare of the local community.",
  "sentence_translation": "她非常關心當地社區的福祉。",
  "prepositions": ["about"]
}, {
  "vocabulary": "concern",
  "part_of_speech": "verb",
  "phonetic_transcription": "/kənˈsɜːrn/",
  "definition": "涉及, 關係到",
  "synonyms": ["involve", "pertain to"],
  "antonyms": ["exclude", "irrelevant"],
  "related_words": ["connection", "significance"],
  "example_sentence": "This new policy concerns all employees in the company.",
  "sentence_translation": "這項新政策涉及公司所有員工。",
  "prepositions": [""]
}]
`
```