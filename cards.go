package main

import (
	"os"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"Finnegan/core/bus"
	"Finnegan/core/watcher"

	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"
)

type CardType string

const (
	CardTypeLocal   CardType = "local" // Saved by user, permanent
	CardTypeBabel   CardType = "babel" // Discovered by Bugu, temporary
	CardTypeThought CardType = "thought"
	CardTypeChatLog CardType = "chat_log"
	CardTypeNote    CardType = "note"
	CardTypeCurio   CardType = "curio"
)

// Card defines the core data model for a consciousness stream piece
type Card struct {
	ID             string              `json:"id"`
	Type           CardType            `json:"type"`
	Origin         string              `json:"origin,omitempty"`
	Title          string              `json:"title"`
	Content        string              `json:"content"`
	Images         []string            `json:"images,omitempty"`
	SourceURL      string              `json:"sourceUrl,omitempty"`
	OriginalText   string              `json:"originalText,omitempty"`
	TranslatedText string              `json:"translatedText,omitempty"`
	PublishedAt    string              `json:"publishedAt,omitempty"`
	AuthorName     string              `json:"authorName,omitempty"`
	AuthorAvatar   string              `json:"authorAvatar,omitempty"`
	EchoOf         string              `json:"echoOf,omitempty"`
	MatchedAnchor  string              `json:"matchedAnchor,omitempty"`
	EchoReason     string              `json:"echoReason,omitempty"`
	UserComment    string              `json:"userComment,omitempty"`
	ReplyCount     int                 `json:"replyCount,omitempty"`
	RepostCount    int                 `json:"repostCount,omitempty"`
	LikeCount      int                 `json:"likeCount,omitempty"`
	CreatedAt      time.Time           `json:"createdAt"`
	Participants   []string            `json:"participants,omitempty"`
	StartedAt      *time.Time          `json:"startedAt,omitempty"`
	EndedAt        *time.Time          `json:"endedAt,omitempty"`
	FilePath       string              `json:"filePath,omitempty"`
	Keywords       map[string][]string `json:"keywords,omitempty"`
	Tags           []string            `json:"tags,omitempty"`

	// Curio Fields
	Brand          string              `json:"brand,omitempty"`
	Platform       string              `json:"platform,omitempty"`
	Price          string              `json:"price,omitempty"`
	Intent         string              `json:"intent,omitempty"`
	Availability   string              `json:"availability,omitempty"`
	SnapshotPath   string              `json:"snapshotPath,omitempty"`
	ScreenshotPath string              `json:"screenshotPath,omitempty"`
}

// TempCardTTL defines the maximum macro in-memory lifetime for temporary network cards (24 hours).
const TempCardTTL = 24 * time.Hour

// CardManager handles the lifecycle of permanent and temporary cards
type CardManager struct {
	tempCards  map[string]Card
	mutex      sync.RWMutex
	stopTicker chan struct{}
}

func NewCardManager() *CardManager {
	cm := &CardManager{
		tempCards:  make(map[string]Card),
		stopTicker: make(chan struct{}),
	}
	cm.startCleanupTicker(10 * time.Minute)
	return cm
}

func (m *CardManager) startCleanupTicker(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.CleanExpiredCards()
			case <-m.stopTicker:
				return
			}
		}
	}()
}

// Close gracefully terminates background cleanup routines
func (m *CardManager) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.stopTicker != nil {
		select {
		case <-m.stopTicker:
		default:
			close(m.stopTicker)
		}
	}
}

// CleanExpiredCards removes all temporary cards that have exceeded TempCardTTL.
// Returns the count of purged cards.
func (m *CardManager) CleanExpiredCards() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	removed := 0
	for id, c := range m.tempCards {
		if !c.CreatedAt.IsZero() && now.Sub(c.CreatedAt) > TempCardTTL {
			delete(m.tempCards, id)
			removed++
		}
	}
	return removed
}

// EvictTempCard removes a specific temporary card by ID.
// Returns true if the card was found and deleted.
func (m *CardManager) EvictTempCard(cardID string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.tempCards[cardID]; exists {
		delete(m.tempCards, cardID)
		return true
	}
	return false
}

// InjectBabelCards receives raw discovered data from Bugu and converts them to temporary cards
func (m *CardManager) InjectBabelCards(cards []Card) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for _, c := range cards {
		c.Type = CardTypeBabel // Force type for safety
		if c.CreatedAt.IsZero() {
			c.CreatedAt = now
		}
		m.tempCards[c.ID] = c
	}
}

// GetTempCards returns all currently discovered network cards that have not expired.
// It also lazily cleans any expired cards encountered.
func (m *CardManager) GetTempCards() []Card {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	cards := make([]Card, 0, len(m.tempCards))
	for id, c := range m.tempCards {
		if !c.CreatedAt.IsZero() && now.Sub(c.CreatedAt) > TempCardTTL {
			delete(m.tempCards, id)
			continue
		}
		cards = append(cards, c)
	}
	return cards
}

// InjectChatLogCard receives a parsed chat log card and stores it in volatile memory
func (m *CardManager) InjectChatLogCard(c Card) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c.Type = CardTypeChatLog
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	m.tempCards[c.ID] = c
}

// PromoteToLocal converts a temporary card to a permanent Local card while losslessly preserving
// its specific type (babel, curio, chat_log, note, thought, etc.) and complete Frontmatter metadata.
func (m *CardManager) PromoteToLocal(cardID string, home string) (*Card, error) {
	return m.PromoteToLocalWithOpts(cardID, home, "", "")
}

// PromoteToLocalWithOpts converts a temporary card to a permanent Local card with custom comment
// and optional echoOf mother card binding override.
func (m *CardManager) PromoteToLocalWithOpts(cardID string, home string, comment string, echoOf string) (*Card, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	card, exists := m.tempCards[cardID]
	if !exists {
		return nil, errors.New("temporary card not found or already saved")
	}

	// 1. Preserve original card type
	// If type is empty, "radar", or "local", normalize to CardTypeThought.
	// Otherwise preserve the exact type (e.g. CardTypeBabel, CardTypeCurio, CardTypeChatLog, CardTypeNote, CardTypeThought).
	targetType := card.Type
	if targetType == "" || targetType == "radar" || targetType == CardTypeLocal {
		targetType = CardTypeThought
	}
	card.Type = targetType

	// Handle comment override
	if strings.TrimSpace(comment) != "" {
		card.UserComment = strings.TrimSpace(comment)
	}

	// Handle echoOf override
	// "NONE" explicitly clears mother card binding.
	// Non-empty string overrides mother card binding.
	if echoOf == "NONE" {
		card.EchoOf = ""
	} else if strings.TrimSpace(echoOf) != "" {
		card.EchoOf = strings.TrimSpace(echoOf)
	}

	now := time.Now()
	newID := ulid.Make().String()
	card.ID = newID

	targetDir := filepath.Join(home, "Scattering")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, err
	}
	
	slugSrc := card.Title
	if slugSrc == "" {
		slugSrc = card.Content
	}
	slug := sanitizeSlug(slugSrc)
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), slug)
	
	var targetPath string
	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}
	if targetPath == "" {
		return nil, errors.New("could not find a unique filename")
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", newID))
	sb.WriteString(fmt.Sprintf("type: %s\n", card.Type))
	if card.Title != "" {
		sb.WriteString(fmt.Sprintf("title: %q\n", card.Title))
	}
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))

	sourceName := "local"
	if card.Origin != "" {
		sourceName = card.Origin
	}
	sb.WriteString(fmt.Sprintf("source: %s\n", sourceName))

	if card.SourceURL != "" {
		sb.WriteString(fmt.Sprintf("source_url: %s\n", card.SourceURL))
	}
	if card.AuthorName != "" {
		sb.WriteString(fmt.Sprintf("author_name: %q\n", card.AuthorName))
	}
	if card.AuthorAvatar != "" {
		sb.WriteString(fmt.Sprintf("author_avatar: %s\n", card.AuthorAvatar))
	}
	if card.PublishedAt != "" {
		sb.WriteString(fmt.Sprintf("published_at: %s\n", card.PublishedAt))
	}
	if card.OriginalText != "" && card.OriginalText != card.Content {
		if origBytes, err := yaml.Marshal(card.OriginalText); err == nil {
			sb.WriteString("original_text: " + string(origBytes))
		}
	}
	if card.TranslatedText != "" && card.TranslatedText != card.Content {
		if transBytes, err := yaml.Marshal(card.TranslatedText); err == nil {
			sb.WriteString("translated_text: " + string(transBytes))
		}
	}
	if card.EchoOf != "" {
		sb.WriteString(fmt.Sprintf("echo_of: %s\n", card.EchoOf))
	}
	if card.MatchedAnchor != "" {
		sb.WriteString(fmt.Sprintf("matched_anchor: %q\n", card.MatchedAnchor))
	}
	if card.EchoReason != "" {
		sb.WriteString(fmt.Sprintf("echo_reason: %q\n", card.EchoReason))
	}
	if card.UserComment != "" {
		sb.WriteString(fmt.Sprintf("user_comment: %q\n", card.UserComment))
	}
	if card.ReplyCount > 0 {
		sb.WriteString(fmt.Sprintf("reply_count: %d\n", card.ReplyCount))
	}
	if card.RepostCount > 0 {
		sb.WriteString(fmt.Sprintf("repost_count: %d\n", card.RepostCount))
	}
	if card.LikeCount > 0 {
		sb.WriteString(fmt.Sprintf("like_count: %d\n", card.LikeCount))
	}

	// ChatLog-specific dialogue metadata
	if len(card.Participants) > 0 {
		sb.WriteString(fmt.Sprintf("participants: [%s]\n", strings.Join(card.Participants, ", ")))
	}
	if card.StartedAt != nil {
		sb.WriteString(fmt.Sprintf("started_at: %s\n", card.StartedAt.Format(time.RFC3339)))
	}
	if card.EndedAt != nil {
		sb.WriteString(fmt.Sprintf("ended_at: %s\n", card.EndedAt.Format(time.RFC3339)))
	}

	// Curio-specific specimen metadata
	if card.Brand != "" {
		sb.WriteString(fmt.Sprintf("brand: %q\n", card.Brand))
	}
	if card.Platform != "" {
		sb.WriteString(fmt.Sprintf("platform: %q\n", card.Platform))
	}
	if card.Price != "" {
		sb.WriteString(fmt.Sprintf("price: %q\n", card.Price))
	}
	if card.Intent != "" {
		sb.WriteString(fmt.Sprintf("intent: %q\n", card.Intent))
	}
	if card.Availability != "" {
		sb.WriteString(fmt.Sprintf("availability: %q\n", card.Availability))
	}
	if card.SnapshotPath != "" {
		sb.WriteString(fmt.Sprintf("snapshotPath: %s\n", card.SnapshotPath))
	}
	if card.ScreenshotPath != "" {
		sb.WriteString(fmt.Sprintf("screenshotPath: %s\n", card.ScreenshotPath))
	}

	// Tags & Keywords
	if len(card.Tags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range card.Tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	if len(card.Keywords) > 0 {
		if kwBytes, err := yaml.Marshal(card.Keywords); err == nil {
			sb.WriteString("keywords:\n")
			lines := strings.Split(strings.TrimSpace(string(kwBytes)), "\n")
			for _, line := range lines {
				sb.WriteString("  " + line + "\n")
			}
		}
	}

	sb.WriteString("publish: false\n")

	if len(card.Images) > 0 {
		sb.WriteString("images:\n")
		for _, img := range card.Images {
			sb.WriteString(fmt.Sprintf("  - %s\n", img))
		}
	}
	sb.WriteString("---\n\n")

	// Body content
	if card.Type != CardTypeChatLog && card.Title != "" {
		trimmed := strings.TrimSpace(card.Content)
		if !strings.HasPrefix(trimmed, "# ") {
			sb.WriteString(fmt.Sprintf("# %s\n\n", card.Title))
		}
	}
	sb.WriteString(card.Content)
	sb.WriteString("\n")

	err := watcher.GetWatcher().WriteFile(targetPath, []byte(sb.String()), "promote")
	if err != nil {
		return nil, err
	}

	card.FilePath = targetPath

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:        newID,
			FilePath:     targetPath,
			Text:         card.Content,
			CapturedAt:   now,
			Images:       card.Images,
			Title:        card.Title,
			Type:         string(card.Type),
			Participants: card.Participants,
			StartedAt:    card.StartedAt,
			EndedAt:      card.EndedAt,
		},
	})

	delete(m.tempCards, cardID)

	return &card, nil
}

// ClearTempCards allows manually flushing the volatile memory
func (m *CardManager) ClearTempCards() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tempCards = make(map[string]Card)
}
