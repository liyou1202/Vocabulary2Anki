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
2. 如果有多個字義則回傳陣列
3. 同義字、反義字、相關字（最多2個nullable)
4. 相關字也要標註詞性
5. 例句、例句翻譯（繁中)
6. 常搭配的介係詞 寫入 ["of",...] ，若無則回傳 []
7. 同義, 反義, 相關, 例句以多使用多益單字為主
僅回傳 JSON，範例：
[{
  "vocabulary": "history",
  "part_of_speech": "noun",
  "phonetic_transcription": "/ˈhɪstəri/",
  "definition": "歷史, 歷史事件",
  "synonyms": ["past", "chronicle"],
  "antonyms": ["future", "present"],
  "related_words": ["historian(noun)", "historic(.adj)"],
  "example_sentence": "The study **of** **history** helps us understand the world.",
  "sentence_translation": "歷史的研究幫助我們理解世界。",
  "prepositions": ["of"]
}, {
  "vocabulary": "history",
  "part_of_speech": "noun",
  "phonetic_transcription": "/ˈhɪstəri/",
  "definition": "過去的經歷, 歷史紀錄",
  "synonyms": ["record", "story"],
  "antonyms": ["future", "novelty"],
  "related_words": ["historian"(noun), "historic(.adj)"],
  "example_sentence": "The **history** **of** this city dates back to the 16th century.",
  "sentence_translation": "這座城市的歷史可以追溯到16世紀。",
  "prepositions": ["of"]
}]
`
```