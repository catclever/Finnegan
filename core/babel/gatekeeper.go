package babel

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	coremodels "Finnegan/core/models"
)

var (
	hashtagRegex = regexp.MustCompile(`#([\p{L}\p{N}_\-]+)`)
	spamAccountKeywords = []string{
		"bot", "airdrop", "crypto", "casino", "giveaway", "promot", "affiliate",
		"onlyfans", "porn", "sex", "escort", "followback", "f4f", "follow4follow",
	}
)

// RawBabelInput represents the uncurated incoming data from any source.
type RawBabelInput struct {
	ID             string
	Origin         string
	URL            string
	Title          string
	RawMarkdown    string
	OriginalText   string
	TranslatedText string
	AuthorName     string
	AuthorAvatar   string
	PublishedAt    string
	EchoOf         string
	MatchedAnchor  string
	EchoReason     string
	Keywords       map[string][]string
	SearchKeywords []string
	ReplyCount     int
	RepostCount    int
	LikeCount      int
}

// Gatekeeper defines the interface for origin-specific gating and resonance filtering.
type Gatekeeper interface {
	Origin() string
	ValidateAndCurate(ctx context.Context, input RawBabelInput) (*coremodels.BabelCard, error)
}

// BuguGatekeeper enforces strict 4-dimensional anchor resonance and long-form markdown quality.
type BuguGatekeeper struct {
	Translator *UniversalTranslator
}

func NewBuguGatekeeper(translator *UniversalTranslator) *BuguGatekeeper {
	return &BuguGatekeeper{Translator: translator}
}

func (g *BuguGatekeeper) Origin() string {
	return "bugu"
}

func (g *BuguGatekeeper) ValidateAndCurate(ctx context.Context, input RawBabelInput) (*coremodels.BabelCard, error) {
	origText := strings.TrimSpace(input.OriginalText)
	if origText == "" {
		return nil, nil // Discard empty
	}

	// 1. Universal spam & quality check
	if isSpam, reason := UniversalSpamFilter(origText); isSpam {
		return nil, fmt.Errorf("bugu gatekeeper rejected: %s", reason)
	}

	// 2. Strict 4-dimensional anchor resonance check:
	// Must have a meaningful MatchedAnchor
	matchedAnchor := strings.TrimSpace(input.MatchedAnchor)
	if matchedAnchor == "" {
		return nil, nil // Discard unresonated card
	}

	// 3. Translation via Universal Base if needed
	transText := strings.TrimSpace(input.TranslatedText)
	if transText == "" && g.Translator != nil {
		var err error
		transText, err = g.Translator.TranslateIfNeeded(ctx, origText)
		if err != nil {
			// Non-fatal: keep going with original text
			transText = ""
		}
	}

	content := origText
	if transText != "" && transText != origText {
		content = transText
	}

	title := input.Title
	if title == "" {
		title = "Babel Echo"
	}

	return &coremodels.BabelCard{
		ID:             input.ID,
		Origin:         "bugu",
		Title:          title,
		Content:        content,
		SourceURL:      input.URL,
		OriginalText:   origText,
		TranslatedText: transText,
		PublishedAt:    input.PublishedAt,
		AuthorName:     input.AuthorName,
		AuthorAvatar:   input.AuthorAvatar,
		EchoOf:         input.EchoOf,
		MatchedAnchor:  matchedAnchor,
		EchoReason:     input.EchoReason,
		ReplyCount:     input.ReplyCount,
		RepostCount:    input.RepostCount,
		LikeCount:      input.LikeCount,
		CreatedAt:      time.Now(),
	}, nil
}

// BlueskyGatekeeper enforces social context resonance, hashtag/keyword anchor extraction, and spam account filtering.
type BlueskyGatekeeper struct {
	Translator *UniversalTranslator
}

func NewBlueskyGatekeeper(translator *UniversalTranslator) *BlueskyGatekeeper {
	return &BlueskyGatekeeper{Translator: translator}
}

func (g *BlueskyGatekeeper) Origin() string {
	return "atproto"
}

func (g *BlueskyGatekeeper) isSpamAccount(handle, authorName string) bool {
	combined := strings.ToLower(handle + " " + authorName)
	for _, kw := range spamAccountKeywords {
		if strings.Contains(combined, kw) {
			return true
		}
	}
	return false
}

func (g *BlueskyGatekeeper) extractAnchor(text string, input RawBabelInput) (string, string) {
	// If explicit matched anchor was already set
	if strings.TrimSpace(input.MatchedAnchor) != "" {
		return strings.TrimSpace(input.MatchedAnchor), input.EchoReason
	}

	lowerText := strings.ToLower(text)

	// 1. Check against 4-dimensional structured keywords if available
	if len(input.Keywords) > 0 {
		for _, words := range input.Keywords {
			for _, w := range words {
				wClean := strings.TrimSpace(w)
				if len([]rune(wClean)) >= 2 && strings.Contains(lowerText, strings.ToLower(wClean)) {
					return wClean, "命中心流关键词"
				}
			}
		}
	}

	// 2. Check against search keywords
	for _, kw := range input.SearchKeywords {
		kwClean := strings.TrimSpace(kw)
		if len([]rune(kwClean)) >= 2 && strings.Contains(lowerText, strings.ToLower(kwClean)) {
			return kwClean, "关键词语义回响"
		}
	}

	// 3. Extract prominent hashtags in the post
	matches := hashtagRegex.FindAllStringSubmatch(text, -1)
	if len(matches) > 0 && len(matches[0]) > 1 {
		tag := matches[0][1]
		if len([]rune(tag)) >= 2 {
			return "#" + tag, "标签语义共鸣"
		}
	}

	// 4. Fallback to primary search keyword if present
	if len(input.SearchKeywords) > 0 && strings.TrimSpace(input.SearchKeywords[0]) != "" {
		return strings.TrimSpace(input.SearchKeywords[0]), "Bluesky 社交共鸣"
	}

	return "", ""
}

func (g *BlueskyGatekeeper) ValidateAndCurate(ctx context.Context, input RawBabelInput) (*coremodels.BabelCard, error) {
	origText := strings.TrimSpace(input.OriginalText)
	if origText == "" {
		return nil, nil // Discard empty
	}

	// 1. Social Spam Account Filter
	if g.isSpamAccount(input.AuthorName, input.AuthorName) {
		return nil, fmt.Errorf("bluesky gatekeeper rejected spam account: %s", input.AuthorName)
	}

	// 2. Universal Spam Filter
	if isSpam, reason := UniversalSpamFilter(origText); isSpam {
		return nil, fmt.Errorf("bluesky gatekeeper rejected spam text: %s", reason)
	}

	// 3. Lightweight Anchor Extraction
	matchedAnchor, echoReason := g.extractAnchor(origText, input)

	// 4. Universal Base: Translation to Chinese if foreign
	transText := strings.TrimSpace(input.TranslatedText)
	if transText == "" && g.Translator != nil {
		var err error
		transText, err = g.Translator.TranslateIfNeeded(ctx, origText)
		if err != nil {
			transText = ""
		}
	}

	content := origText
	if transText != "" && transText != origText {
		content = transText
	}

	title := input.Title
	if title == "" {
		title = "Bluesky Echo"
	}

	return &coremodels.BabelCard{
		ID:             input.ID,
		Origin:         "atproto",
		Title:          title,
		Content:        content,
		SourceURL:      input.URL,
		OriginalText:   origText,
		TranslatedText: transText,
		PublishedAt:    input.PublishedAt,
		AuthorName:     input.AuthorName,
		AuthorAvatar:   input.AuthorAvatar,
		EchoOf:         input.EchoOf,
		MatchedAnchor:  matchedAnchor,
		EchoReason:     echoReason,
		ReplyCount:     input.ReplyCount,
		RepostCount:    input.RepostCount,
		LikeCount:      input.LikeCount,
		CreatedAt:      time.Now(),
	}, nil
}
