package models

import "time"

// BabelCard represents the distilled, translated information extracted from a webpage.
type BabelCard struct {
	SourceURL      string    `json:"source_url"`
	OriginalText   string    `json:"original_text"`
	TranslatedText string    `json:"translated_text"`
	EchoOf         string    `json:"echo_of,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
