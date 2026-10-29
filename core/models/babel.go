package models

import "time"

// BabelCard represents the universal, distilled intelligence card model
// shared across Bugu (deep web distillation), Bluesky (social pulse echoes),
// and future consciousness stream sources.
type BabelCard struct {
	ID             string    `json:"id,omitempty"`
	Origin         string    `json:"origin,omitempty"`
	Title          string    `json:"title,omitempty"`
	Content        string    `json:"content,omitempty"`
	SourceURL      string    `json:"source_url"`
	OriginalText   string    `json:"original_text"`
	TranslatedText string    `json:"translated_text"`
	PublishedAt    string    `json:"published_at,omitempty"`
	AuthorName     string    `json:"author_name,omitempty"`
	AuthorAvatar   string    `json:"author_avatar,omitempty"`
	EchoOf         string    `json:"echo_of,omitempty"`
	MatchedAnchor  string    `json:"matched_anchor,omitempty"`
	EchoReason     string    `json:"echo_reason,omitempty"`
	ReplyCount     int       `json:"reply_count,omitempty"`
	RepostCount    int       `json:"repost_count,omitempty"`
	LikeCount      int       `json:"like_count,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
