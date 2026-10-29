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

// PromoteToLocal converts a temporary Radar or ChatLog card to a permanent Local card
func (m *CardManager) PromoteToLocal(cardID string, home string) (*Card, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	card, exists := m.tempCards[cardID]
	if !exists {
		return nil, errors.New("temporary card not found or already saved")
	}

	if card.Type != CardTypeChatLog {
		card.Type = CardTypeLocal
	}
	now := time.Now()
	newID := ulid.Make().String()
	card.ID = newID

	targetDir := filepath.Join(home, "Scattering")
	
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
	if card.Type == CardTypeChatLog {
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("id: %s\n", newID))
		sb.WriteString("type: chat_log\n")
		if card.Title != "" {
			sb.WriteString(fmt.Sprintf("title: %s\n", card.Title))
		}
		if len(card.Participants) > 0 {
			sb.WriteString(fmt.Sprintf("participants: [%s]\n", strings.Join(card.Participants, ", ")))
		}
		if card.StartedAt != nil {
			sb.WriteString(fmt.Sprintf("started_at: %s\n", card.StartedAt.Format(time.RFC3339)))
		}
		if card.EndedAt != nil {
			sb.WriteString(fmt.Sprintf("ended_at: %s\n", card.EndedAt.Format(time.RFC3339)))
		}
		if card.StartedAt != nil {
			sb.WriteString(fmt.Sprintf("created_at: %s\n", card.StartedAt.Format(time.RFC3339)))
		} else {
			sb.WriteString(fmt.Sprintf("created_at: %s\n", now.Format(time.RFC3339)))
		}
		if len(card.Images) > 0 {
			sb.WriteString("images:\n")
			for _, img := range card.Images {
				sb.WriteString(fmt.Sprintf("  - %s\n", img))
			}
		}
		sb.WriteString("---\n\n")
		sb.WriteString(card.Content)
		sb.WriteString("\n")
	} else {
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("id: %s\n", newID))
		sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
		sourceName := "bugu"
		if card.Origin != "" {
			sourceName = card.Origin
		}
		sb.WriteString(fmt.Sprintf("source: %s\n", sourceName))
		if card.SourceURL != "" {
			sb.WriteString(fmt.Sprintf("source_url: %s\n", card.SourceURL))
		}
		if card.EchoOf != "" {
			sb.WriteString(fmt.Sprintf("echo_of: %s\n", card.EchoOf))
		}
		sb.WriteString("publish: false\n")
		if len(card.Images) > 0 {
			sb.WriteString("images:\n")
			for _, img := range card.Images {
				sb.WriteString(fmt.Sprintf("  - %s\n", img))
			}
		}
		sb.WriteString("---\n\n")
		if card.Title != "" {
			sb.WriteString(fmt.Sprintf("# %s\n\n", card.Title))
		}
		sb.WriteString(card.Content)
		sb.WriteString("\n")
	}

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
