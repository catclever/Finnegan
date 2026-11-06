package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/keywords"
	"Finnegan/core/module"
	"Finnegan/core/watcher"
	_ "Finnegan/modules/atproto"
	_ "Finnegan/modules/bugu"
	_ "Finnegan/modules/curio"
	_ "Finnegan/modules/diary"
	_ "Finnegan/modules/geo"
	"path/filepath"
)

type ImagePayload struct {
	Data string `json:"data"`
	Name string `json:"name,omitempty"`
	Ext  string `json:"ext,omitempty"`
}

type ChatLogCaptureResult struct {
	DocID        string     `json:"docId"`
	Title        string     `json:"title"`
	Participants []string   `json:"participants,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	EndedAt      *time.Time `json:"endedAt,omitempty"`
	Images       []string   `json:"images,omitempty"`
}

type App struct {
	ctx           context.Context
	CardManager   *CardManager
	KeywordEngine *keywords.KeywordEngine
	Home          string
}

func NewApp() *App {
	a := &App{}
	cfg, home, err := config.LoadConfig()
	if err == nil {
		a.Home = home
		llmCfg := a.getLLMConfig()
		kwCfg := keywords.DefaultConfig()
		if cfg != nil {
			if node, exists := cfg.RawModules["keywords"]; exists {
				_ = node.Decode(&kwCfg)
			}
			a.KeywordEngine = keywords.NewKeywordEngine(llmCfg, kwCfg, cfg.Jina)
		} else {
			a.KeywordEngine = keywords.NewKeywordEngine(llmCfg, kwCfg)
		}
	}
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, home, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Critical Error: Failed to load core configuration: %v", err)
	}
	a.Home = home

	// Discover and register external plugins from ~/.finnegan/plugins
	if home != "" {
		pluginsDir := filepath.Join(home, ".finnegan", "plugins")
		if _, err := module.LoadExternalPlugins(pluginsDir); err != nil {
			log.Printf("Warning: Failed to load external plugins from %s: %v", pluginsDir, err)
		}
	}

	// Init modules
	hostAPI := &appHostAPI{app: a}
	reg := module.GetRegistry()
	for name, m := range reg {
		if cfg != nil && cfg.RawModules != nil {
			if node, exists := cfg.RawModules[name]; exists {
				if target := m.ConfigTarget(); target != nil {
					_ = node.Decode(target)
				}
			}
		}
		mCtx := &module.Context{
			LLMs:    cfg.LLMs,
			Storage: nil,
			Watcher: nil,
			Host:    hostAPI,
		}
		if err := m.Init(mCtx); err != nil {
			log.Printf("[%s] module init failed: %v", name, err)
		}
	}

	a.CardManager = NewCardManager()

	_ = bus.GetBus()

	wStorage := &WatcherStorage{ctx: a.ctx}
	a.setupWatcherCallbacks(watcher.GetWatcher(), wStorage)

	watchDirs := config.DeriveWatchDirs(home, cfg.ExternalProjects)
	err = watcher.GetWatcher().Initialize(watchDirs, cfg.DebounceMs, wStorage, home)
	if err != nil {
		log.Printf("Failed to initialize watcher: %v", err)
	}

	// Initialize / refresh KeywordEngine
	llmCfg := a.getLLMConfig()
	kwCfg := keywords.DefaultConfig()
	if cfg != nil {
		if node, exists := cfg.RawModules["keywords"]; exists {
			_ = node.Decode(&kwCfg)
		}
	}
	if a.KeywordEngine == nil {
		if cfg != nil {
			a.KeywordEngine = keywords.NewKeywordEngine(llmCfg, kwCfg, cfg.Jina)
		} else {
			a.KeywordEngine = keywords.NewKeywordEngine(llmCfg, kwCfg)
		}
	} else {
		a.KeywordEngine.LLMConfig = llmCfg
		a.KeywordEngine.Config = kwCfg
		if cfg != nil {
			a.KeywordEngine.JinaConfig = cfg.Jina
		}
	}

	a.setupEventSubscriptions()
}

func (a *App) GetKeywordEngine() *keywords.KeywordEngine {
	return a.KeywordEngine
}

func (a *App) ExtractCardKeywords(filePath string, docID string, text string) {
	if a.KeywordEngine != nil {
		a.KeywordEngine.ProcessCard(filePath, docID, text)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.CardManager != nil {
		a.CardManager.Close()
	}
	watcher.GetWatcher().Close()
	bus.GetBus().Close()
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) getLLMConfig() module.LLMConfig {
	cfg, _, err := config.LoadConfig()
	if err == nil && cfg != nil {
		if def, ok := cfg.LLMs["default"]; ok {
			return def
		}
	}
	return module.LLMConfig{
		Provider:     "gemini",
		APIKey:       os.Getenv("GEMINI_API_KEY"),
		DefaultModel: "gemini-3.8-flash",
	}
}
