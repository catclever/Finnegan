package webcapture

import (
	"context"
	"strings"
	"time"

	"Finnegan/core/chatlog"
	"Finnegan/core/config"

	"github.com/oklog/ulid/v2"
)

// ProcessedCard encapsulates all processed information of a web capture.
type ProcessedCard struct {
	ID           string
	Type         CardType
	Title        string
	Content      string // Markdown body (coalesced dialogue or note text)
	FullMarkdown string // Full markdown file content with YAML frontmatter
	SourceURL    string
	Participants []string
	StartedAt    *time.Time
	EndedAt      *time.Time
	CreatedAt    time.Time
}

// Pipeline orchestrates fetching and processing of web captures.
type Pipeline struct {
	Fetcher *Fetcher
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(cfg ...config.JinaConfig) *Pipeline {
	return &Pipeline{
		Fetcher: NewFetcher(cfg...),
	}
}

// WithFetcher sets a custom Fetcher for testing or custom configurations.
func (p *Pipeline) WithFetcher(fetcher *Fetcher) *Pipeline {
	p.Fetcher = fetcher
	return p
}

// Capture fetches targetURL via Jina Reader and parses it into a ProcessedCard.
func (p *Pipeline) Capture(ctx context.Context, targetURL string) (*ProcessedCard, error) {
	rawMarkdown, err := p.Fetcher.Fetch(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	docID := ulid.Make().String()
	now := time.Now()

	return ProcessContent(targetURL, rawMarkdown, docID, now)
}

// ProcessContent transforms raw markdown into a structured ProcessedCard according to routing rules.
func ProcessContent(urlStr string, rawMarkdown string, docID string, createdAt time.Time) (*ProcessedCard, error) {
	if IsGeminiShare(urlStr, rawMarkdown) {
		rawMarkdown = PreprocessGeminiShare(rawMarkdown)
	}

	cardType := Route(urlStr, rawMarkdown)

	if cardType == CardTypeChatLog {
		title, remaining := ExtractTitle(rawMarkdown, "")
		cleanContent := CleanJinaHeaders(remaining)

		parsed, err := chatlog.Parse(cleanContent)
		var body string
		var participants []string
		var startedAt, endedAt *time.Time

		if err == nil && parsed != nil {
			if title == "" && parsed.UserTitle != "" {
				title = parsed.UserTitle
			}
			body = parsed.Coalesced
			participants = parsed.Participants
			startedAt = parsed.StartedAt
			endedAt = parsed.EndedAt
		} else {
			// If chatlog.Parse fails (e.g. domain rule matched but speakers don't match strict colons),
			// preserve clean markdown body
			body = cleanContent
		}

		if title == "" {
			title = "Chat Log"
		}

		fullMarkdown := FormatChatLogCard(docID, title, urlStr, body, parsed, createdAt)

		return &ProcessedCard{
			ID:           docID,
			Type:         CardTypeChatLog,
			Title:        title,
			Content:      body,
			FullMarkdown: fullMarkdown,
			SourceURL:    urlStr,
			Participants: participants,
			StartedAt:    startedAt,
			EndedAt:      endedAt,
			CreatedAt:    createdAt,
		}, nil
	}

	// Default: CardTypeNote
	title, remaining := ExtractTitle(rawMarkdown, "")
	cleanBody := CleanJinaHeaders(remaining)

	// Avoid duplicate # Title at top of cleanBody if it matches title
	trimmed := strings.TrimSpace(cleanBody)
	if strings.HasPrefix(trimmed, "# ") {
		lines := strings.Split(trimmed, "\n")
		firstH1 := strings.TrimSpace(strings.TrimPrefix(lines[0], "# "))
		if strings.EqualFold(firstH1, title) {
			cleanBody = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}
	} else {
		cleanBody = trimmed
	}

	if title == "" {
		title = "Web Note"
	}

	fullMarkdown := FormatNoteCard(docID, title, urlStr, cleanBody, createdAt)

	return &ProcessedCard{
		ID:           docID,
		Type:         CardTypeNote,
		Title:        title,
		Content:      cleanBody,
		FullMarkdown: fullMarkdown,
		SourceURL:    urlStr,
		CreatedAt:    createdAt,
	}, nil
}
