package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/chatlog"
	"Finnegan/core/config"
	"Finnegan/core/watcher"

	"github.com/oklog/ulid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ParseChatLog parses raw pasted chat log text and returns parsed metadata and coalesced text.
func (a *App) ParseChatLog(rawText string) (*chatlog.ParsedChatLog, error) {
	return chatlog.Parse(rawText)
}

// DeriveChatLogTitle derives a concise title and participants using LLM for the given chat log dialogue.
func (a *App) DeriveChatLogTitle(dialogue string, participants []string) (*chatlog.TitleResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	llmCfg := a.getLLMConfig()
	return chatlog.DeriveTitle(ctx, llmCfg, dialogue, participants)
}

// ProcessChatLog creates a temporary (volatile) chat_log card in memory and emits an event for the CosmicStage.
func (a *App) ProcessChatLog(rawText string) (*Card, error) {
	parsed, err := chatlog.Parse(rawText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chat log: %w", err)
	}

	var title string
	if parsed.UserTitle != "" {
		title = parsed.UserTitle
	} else {
		titleRes, _ := a.DeriveChatLogTitle(parsed.Coalesced, parsed.Participants)
		if titleRes != nil {
			title = titleRes.Title
			if len(titleRes.Participants) > 0 {
				parsed.Participants = titleRes.Participants
			}
		}
	}
	if title == "" {
		title = chatlog.DeriveFallbackTitle(parsed.Coalesced, parsed.Participants)
	}

	now := time.Now()
	createdAt := now
	if parsed.StartedAt != nil {
		createdAt = *parsed.StartedAt
	}

	cardID := fmt.Sprintf("chatlog_%s", ulid.Make().String())
	card := Card{
		ID:           cardID,
		Type:         CardTypeChatLog,
		Title:        title,
		Content:      parsed.Coalesced,
		Participants: parsed.Participants,
		StartedAt:    parsed.StartedAt,
		EndedAt:      parsed.EndedAt,
		CreatedAt:    createdAt,
	}

	if a.CardManager != nil {
		a.CardManager.InjectChatLogCard(card)
	}

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "chatlog:card_spawned", map[string]interface{}{
			"id":           card.ID,
			"type":         string(card.Type),
			"title":        card.Title,
			"content":      card.Content,
			"participants": card.Participants,
			"startedAt":    card.StartedAt,
			"endedAt":      card.EndedAt,
			"createdAt":    card.CreatedAt,
		})
	}

	return &card, nil
}

// CaptureChatLogWithImages directly parses and saves a chat log card with image attachments to Scattering and assets.
func (a *App) CaptureChatLogWithImages(rawText string, images []ImagePayload) (*ChatLogCaptureResult, error) {
	parsed, err := chatlog.Parse(rawText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chat log: %w", err)
	}

	var title string
	if parsed.UserTitle != "" {
		title = parsed.UserTitle
	} else {
		titleRes, _ := a.DeriveChatLogTitle(parsed.Coalesced, parsed.Participants)
		if titleRes != nil {
			title = titleRes.Title
			if len(titleRes.Participants) > 0 {
				parsed.Participants = titleRes.Participants
			}
		}
	}
	if title == "" {
		title = chatlog.DeriveFallbackTitle(parsed.Coalesced, parsed.Participants)
	}

	home := a.Home
	if home == "" {
		_, h, err := config.LoadConfig()
		if err == nil {
			home = h
			a.Home = h
		}
	}
	if home == "" {
		return nil, errors.New("home directory not configured")
	}

	savedImageNames, err := a.saveAssetImages(images)
	if err != nil {
		return nil, fmt.Errorf("failed to save asset images: %w", err)
	}
	parsed.Images = savedImageNames

	targetDir := filepath.Join(home, "Scattering")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}
	now := time.Now()
	slugSrc := title
	if slugSrc == "" {
		slugSrc = "chat_log"
	}
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), sanitizeSlug(slugSrc))
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

	docID := ulid.Make().String()
	contentStr := chatlog.FormatCardMarkdown(docID, title, parsed, savedImageNames)

	err = watcher.GetWatcher().WriteFile(targetPath, []byte(contentStr), "capture")
	if err != nil {
		return nil, err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:        docID,
			FilePath:     targetPath,
			Text:         parsed.Coalesced,
			CapturedAt:   now,
			Images:       savedImageNames,
			Title:        title,
			Type:         "chat_log",
			Participants: parsed.Participants,
			StartedAt:    parsed.StartedAt,
			EndedAt:      parsed.EndedAt,
		},
	})

	return &ChatLogCaptureResult{
		DocID:        docID,
		Title:        title,
		Participants: parsed.Participants,
		StartedAt:    parsed.StartedAt,
		EndedAt:      parsed.EndedAt,
		Images:       savedImageNames,
	}, nil
}

// CaptureChatLog directly parses and saves a chat log card as a permanent Markdown file in Scattering.
func (a *App) CaptureChatLog(rawText string) (*ChatLogCaptureResult, error) {
	return a.CaptureChatLogWithImages(rawText, nil)
}
