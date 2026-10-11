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

	"Finnegan/bugu"
	"Finnegan/bugu/api"
	buguconfig "Finnegan/bugu/config"
	"Finnegan/bugu/models"
	
	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/watcher"

	"github.com/oklog/ulid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	buguConfig  *buguconfig.Config
	sysConfig   *config.Config
	Home        string
	CardManager *CardManager
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	configPaths := []string{
		"bugu/radar_targets.yml",
		"radar_targets.yml",
	}
	for _, p := range configPaths {
		if _, err := os.Stat(p); err == nil {
			cfg, err := buguconfig.LoadConfig(p)
			if err == nil {
				a.buguConfig = cfg
				log.Printf("[Finnegan] Bugu radar config loaded with %d targets from %s", len(cfg.Targets), p)
				break
			}
		}
	}

	cfg, home, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		cfg = &config.Config{ExternalProjects: []string{}, DebounceMs: 300, BuguMaxConcurrency: 1}
	}
	a.sysConfig = cfg
	a.Home = home

	// Delete old config if exists
	if _, err := os.Stat("finnegan.yml"); err == nil {
		os.Remove("finnegan.yml")
		log.Println("Deleted legacy finnegan.yml from repository root.")
	}

	dbPath := filepath.Join(home, ".finnegan", "cache.db")
	db, err := NewRxDBStorage(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	a.CardManager = NewCardManager(db)

	_ = bus.GetBus()

	wStorage := &WatcherStorage{db: a.CardManager.db, ctx: a.ctx}
	watchDirs := config.DeriveWatchDirs(home, cfg.ExternalProjects)
	err = watcher.GetWatcher().Initialize(watchDirs, cfg.DebounceMs, wStorage, home)
	if err != nil {
		log.Printf("Failed to initialize watcher: %v", err)
	}
	
	injectFunc := func(docID string, intent string, results []string) {
		var cards []Card
		for i, res := range results {
			c := Card{
				ID:        fmt.Sprintf("bugu-%s-%d-%d", docID, time.Now().UnixNano(), i),
				Title:     fmt.Sprintf("Radar: %s", intent),
				Content:   res,
				SourceURL: "",
				CreatedAt: time.Now(),
			}
			cards = append(cards, c)
		}
		a.CardManager.InjectRadarCards(cards)
		
		runtime.EventsEmit(a.ctx, "radar:results", map[string]interface{}{
			"docId": docID,
			"count": len(results),
		})
	}
	
	if a.buguConfig == nil {
		log.Printf("[Finnegan] Bugu config is nil, capture listener will not be started")
	} else {
		bl := bugu.NewCaptureListener(a.buguConfig, cfg.BuguMaxConcurrency, injectFunc)
		bl.Start()
	}
}

func (a *App) shutdown(ctx context.Context) {
    watcher.GetWatcher().Close()
	bus.GetBus().Close()
}

func sanitizeSlug(text string) string {
	runes := []rune(text)
	if len(runes) > 20 {
		runes = runes[:20]
	}
	s := string(runes)
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "\\", "")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, "<", "")
	s = strings.ReplaceAll(s, ">", "")
	s = strings.ReplaceAll(s, "|", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "thought"
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
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}
	
	if targetPath == "" {
		return "", errors.New("could not find a unique filename")
	}
	
	docID := ulid.Make().String()
	
	contentStr := fmt.Sprintf("---\nid: %s\ncreated: %s\nupdated: %s\nsource: local\ntags: []\npublish: false\n---\n\n%s\n", 
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
	db  *RxDBStorage
	ctx context.Context
}

func (w *WatcherStorage) SaveNote(id, data string) error {
	err := w.db.BulkSaveDocs("cards", []DocSaveRequest{
		{ID: id, Data: data},
	})
	if err == nil && w.ctx != nil {
		runtime.EventsEmit(w.ctx, "notes:changed", map[string]interface{}{
			"docId": id,
			"op":    "upsert",
		})
	}
	return err
}

func (w *WatcherStorage) DeleteNote(id string) error {
	err := w.db.DeleteDoc("cards", id)
	if err == nil && w.ctx != nil {
		runtime.EventsEmit(w.ctx, "notes:changed", map[string]interface{}{
			"docId": id,
			"op":    "remove",
		})
	}
	return err
}

func (w *WatcherStorage) GetNote(id string) (string, error) {
	return w.db.GetDoc("cards", id)
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) GetRadarTargets() []models.Target {
	if a.buguConfig == nil {
		return []models.Target{}
	}
	return a.buguConfig.Targets
}

func (a *App) RunRadar(intent string) api.RadarResponse {
	if a.buguConfig == nil {
		return api.RadarResponse{
			Status: "error",
			Error:  "Bugu configuration is not loaded",
		}
	}
	return api.ExecuteRadar(a.buguConfig, intent)
}

func (a *App) GetTempCards() []Card {
	return a.CardManager.GetTempCards()
}

func (a *App) PromoteTempCard(cardID string) (*Card, error) {
	return a.CardManager.PromoteToLocal(cardID, a.Home)
}

func (a *App) TestInjectBuguCard(title, content, sourceURL string) Card {
	c := Card{
		ID:        "temp-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Title:     title,
		Content:   content,
		SourceURL: sourceURL,
		CreatedAt: time.Now(),
	}
	a.CardManager.InjectRadarCards([]Card{c})
	return c
}

func (a *App) GetNote(id string) (string, error) {
	return a.CardManager.db.GetDoc("cards", id)
}

func (a *App) ClearTempCards() {
	a.CardManager.ClearTempCards()
}

func (w *WatcherStorage) EmitConflict(id string, paths []string) {
	if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "card:id_conflict", map[string]interface{}{
			"id":    id,
			"paths": paths,
		})
	}
}
