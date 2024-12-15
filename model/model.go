package model

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
	Archived              int      `json:"archived"`
}
