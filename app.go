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
			runtime.EventsEmit(a.ctx, "bugu:card_spawned", event.Data)
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
