package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/module"
	"Finnegan/core/search"
	"Finnegan/core/watcher"
	"Finnegan/modules/bugu"
	"Finnegan/modules/bugu/api"
	"Finnegan/modules/bugu/models"

	"github.com/oklog/ulid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	CardManager *CardManager
	buguModule  *bugu.BuguModule
	Home        string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, home, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Critical Error: Failed to load core configuration: %v", err)
	}
	a.Home = home

	// Init modules
	reg := module.GetRegistry()
	if bModule, ok := reg["bugu"].(*bugu.BuguModule); ok {
		mCtx := &module.Context{
			LLMs:    cfg.LLMs,
			Storage: nil,
			Watcher: nil,
		}
		if err := bModule.Init(mCtx); err != nil {
			log.Printf("Bugu module init failed: %v", err)
		} else {
			a.buguModule = bModule
		}
	}

	a.CardManager = NewCardManager()

	_ = bus.GetBus()

	wStorage := &WatcherStorage{ctx: a.ctx}
	watchDirs := config.DeriveWatchDirs(home, cfg.ExternalProjects)
	err = watcher.GetWatcher().Initialize(watchDirs, cfg.DebounceMs, wStorage, home)
	if err != nil {
		log.Printf("Failed to initialize watcher: %v", err)
	}
	
	injectFunc := func(docID string, intent string, results []string) {
		var cards []Card
		for i, res := range results {
			c := Card{
				ID:             fmt.Sprintf("%s-radar-%d", docID, i),
				Type:           CardTypeBabel,
				Title:          fmt.Sprintf("Distilled Insights #%d", i+1),
				Content:        res,
				EchoOf:         docID,
				CreatedAt:      time.Now(),
			}
			cards = append(cards, c)
		}
		a.CardManager.InjectBabelCards(cards)
	}
	_ = injectFunc

	babelCh := bus.GetBus().Subscribe(bus.EventBabelCardSpawned, "wails-frontend")
	go func() {
		for event := range babelCh {
			var cardID string
			var title, content, origText, transText, srcURL, echoOf string
			var publishedAt, authorName, authorAvatar string

			if bc, ok := event.Data.(models.BabelCard); ok {
				cardID = fmt.Sprintf("babel_%s", ulid.Make().String())
				title = "Babel Echo"
				origText = bc.OriginalText
				transText = bc.TranslatedText
				if transText != "" && transText != origText {
					content = transText
				} else {
					content = origText
				}
				srcURL = bc.SourceURL
				echoOf = bc.EchoOf
				publishedAt = bc.PublishedAt
				authorName = bc.AuthorName
				authorAvatar = bc.AuthorAvatar
			} else if bcp, ok := event.Data.(*models.BabelCard); ok && bcp != nil {
				cardID = fmt.Sprintf("babel_%s", ulid.Make().String())
				title = "Babel Echo"
				origText = bcp.OriginalText
				transText = bcp.TranslatedText
				if transText != "" && transText != origText {
					content = transText
				} else {
					content = origText
				}
				srcURL = bcp.SourceURL
				echoOf = bcp.EchoOf
				publishedAt = bcp.PublishedAt
				authorName = bcp.AuthorName
				authorAvatar = bcp.AuthorAvatar
			}

			if cardID != "" {
				if a.CardManager != nil {
					a.CardManager.InjectBabelCards([]Card{
						{
							ID:             cardID,
							Type:           CardTypeBabel,
							Title:          title,
							Content:        content,
							SourceURL:      srcURL,
							OriginalText:   origText,
							TranslatedText: transText,
							PublishedAt:    publishedAt,
							AuthorName:     authorName,
							AuthorAvatar:   authorAvatar,
							EchoOf:         echoOf,
							CreatedAt:      time.Now(),
						},
					})
				}
				runtime.EventsEmit(a.ctx, "bugu:card_spawned", map[string]interface{}{
					"id":             cardID,
					"type":           "babel",
					"title":          title,
					"content":        content,
					"sourceUrl":      srcURL,
					"originalText":   origText,
					"translatedText": transText,
					"source_url":      srcURL,
					"original_text":   origText,
					"translated_text": transText,
					"publishedAt":    publishedAt,
					"published_at":   publishedAt,
					"authorName":     authorName,
					"author_name":    authorName,
					"authorAvatar":   authorAvatar,
					"author_avatar":  authorAvatar,
					"echoOf":         echoOf,
					"echo_of":        echoOf,
					"createdAt":      time.Now(),
				})
			} else {
				runtime.EventsEmit(a.ctx, "bugu:card_spawned", event.Data)
			}
		}
	}()

	thoughtCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "wails-frontend")
	go func() {
		for event := range thoughtCh {
			runtime.EventsEmit(a.ctx, "thought:captured", event.Data)
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	watcher.GetWatcher().Close()
	bus.GetBus().Close()
}

func sanitizeSlug(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	if len(s) > 20 {
		runes := []rune(s)
		return string(runes[:20])
	}
	return s
}

func (a *App) CaptureThought(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("empty text")
	}
	
	targetDir := filepath.Join(a.Home, "Scattering")

	now := time.Now()
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), sanitizeSlug(text))
	var targetPath string
	
	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.draft.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}
	
	if targetPath == "" {
		return "", errors.New("could not find a unique filename")
	}
	
	docID := ulid.Make().String()
	
	contentStr := fmt.Sprintf("---\nid: %s\ntype: thought\ncreated: %s\nupdated: %s\nsource: local\ntags: []\npublish: false\n---\n\n%s\n", 
		docID, now.Format(time.RFC3339), now.Format(time.RFC3339), text)
	
	err := watcher.GetWatcher().WriteFile(targetPath, []byte(contentStr), "capture")
	if err != nil {
		return "", err
	}

	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      docID,
			FilePath:   targetPath,
			Text:       text,
			CapturedAt: now,
		},
	})
	
	return docID, nil
}

type WatcherStorage struct {
	ctx context.Context
}

func (w *WatcherStorage) SaveNote(id, data string) error {
	if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "notes:changed", map[string]interface{}{
			"docId": id,
			"op":    "upsert",
		})
	}
	return nil
}

func (w *WatcherStorage) DeleteNote(id string) error {
	if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "notes:changed", map[string]interface{}{
			"docId": id,
			"op":    "remove",
		})
	}
	return nil
}

func (w *WatcherStorage) GetNote(id string) (string, error) {
	return "", nil
}

func (w *WatcherStorage) EmitConflict(id string, paths []string) {
	if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "card:id_conflict", map[string]interface{}{
			"id":    id,
			"paths": paths,
		})
	}
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) GetRadarTargets() []models.Target {
	if a.buguModule == nil {
		return []models.Target{}
	}
	return a.buguModule.GetRadarTargets()
}

func (a *App) RunRadar(intent string) api.RadarResponse {
	if a.buguModule == nil {
		return api.RadarResponse{
			Status: "error",
			Error:  "Bugu configuration is not loaded",
		}
	}
	return a.buguModule.RunRadar(intent)
}

func (a *App) GetTempCards() []Card {
	return a.CardManager.GetTempCards()
}

func (a *App) PromoteTempCard(cardID string) (*Card, error) {
	return a.CardManager.PromoteToLocal(cardID, a.Home)
}

func (a *App) GetNote(id string) (string, error) {
	return "", nil
}

func (a *App) ClearTempCards() {
	a.CardManager.ClearTempCards()
}

// SearchLocalCards loads local cards and temporary cards, runs fuzzy search, and returns sorted results.
func (a *App) SearchLocalCards(query string) []Card {
	var rawCards []Card
	rawCards = append(rawCards, a.LoadLocalCards()...)
	rawCards = append(rawCards, a.GetTempCards()...)

	// Deduplicate rawCards by ID and content
	seenID := make(map[string]bool)
	seenContent := make(map[string]bool)
	var uniqueCards []Card
	for _, c := range rawCards {
		trimmedContent := strings.TrimSpace(c.Content)
		if c.ID != "" && seenID[c.ID] {
			continue
		}
		if trimmedContent != "" && seenContent[trimmedContent] {
			continue
		}
		if c.ID != "" {
			seenID[c.ID] = true
		}
		if trimmedContent != "" {
			seenContent[trimmedContent] = true
		}
		uniqueCards = append(uniqueCards, c)
	}

	sCards := make([]search.Card, len(uniqueCards))
	for i, c := range uniqueCards {
		sCards[i] = search.Card{
			ID:             c.ID,
			Type:           string(c.Type),
			Title:          c.Title,
			Content:        c.Content,
			SourceURL:      c.SourceURL,
			OriginalText:   c.OriginalText,
			TranslatedText: c.TranslatedText,
			EchoOf:         c.EchoOf,
			CreatedAt:      c.CreatedAt,
		}
	}

	matched := search.SearchCards(query, sCards)

	cardMap := make(map[string]Card, len(uniqueCards))
	for _, c := range uniqueCards {
		cardMap[c.ID] = c
	}

	resSeenID := make(map[string]bool)
	resSeenContent := make(map[string]bool)
	results := make([]Card, 0, len(matched))
	for _, sc := range matched {
		trimmedContent := strings.TrimSpace(sc.Content)
		if sc.ID != "" && resSeenID[sc.ID] {
			continue
		}
		if trimmedContent != "" && resSeenContent[trimmedContent] {
			continue
		}
		if sc.ID != "" {
			resSeenID[sc.ID] = true
		}
		if trimmedContent != "" {
			resSeenContent[trimmedContent] = true
		}

		if orig, ok := cardMap[sc.ID]; ok {
			results = append(results, orig)
		} else {
			results = append(results, Card{
				ID:             sc.ID,
				Type:           CardType(sc.Type),
				Title:          sc.Title,
				Content:        sc.Content,
				SourceURL:      sc.SourceURL,
				OriginalText:   sc.OriginalText,
				TranslatedText: sc.TranslatedText,
				EchoOf:         sc.EchoOf,
				CreatedAt:      sc.CreatedAt,
			})
		}
	}

	return results
}

// GetRegisteredCommands iterates module.GetRegistry, finds modules implementing CommandProvider, and collects all CommandDefs.
func (a *App) GetRegisteredCommands() []module.CommandDef {
	commands := make([]module.CommandDef, 0)
	for _, m := range module.GetRegistry() {
		if cp, ok := m.(module.CommandProvider); ok {
			commands = append(commands, cp.GetCommands()...)
		}
	}
	return commands
}

// ExecuteModuleCommand dispatches a command to the module that owns cmdID.
func (a *App) ExecuteModuleCommand(cmdID string, params map[string]interface{}) (interface{}, error) {
	for _, m := range module.GetRegistry() {
		if cp, ok := m.(module.CommandProvider); ok {
			for _, cmd := range cp.GetCommands() {
				if cmd.ID == cmdID {
					ctx := a.ctx
					if ctx == nil {
						ctx = context.Background()
					}
					return cp.ExecuteCommand(ctx, cmdID, params)
				}
			}
		}
	}
	return nil, fmt.Errorf("command not found: %s", cmdID)
}
