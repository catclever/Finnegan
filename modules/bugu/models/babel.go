package models

// BabelCard represents the distilled, translated information extracted from a webpage.
type BabelCard struct {
	SourceURL      string `json:"source_url"`
	OriginalText   string `json:"original_text"`
	TranslatedText string `json:"translated_text"`
}
