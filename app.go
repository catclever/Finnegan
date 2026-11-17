package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/keywords"
	"Finnegan/core/module"
	"Finnegan/core/watcher"
	_ "Finnegan/modules/atproto"
	_ "Finnegan/modules/bugu"
	_ "Finnegan/modules/diary"
	_ "Finnegan/modules/geo"
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
	ctx               context.Context
	CardManager       *CardManager
	KeywordEngine     *keywords.KeywordEngine
	Home              string
	reloadMu          sync.Mutex
	pluginWatcherMu   sync.Mutex
	pluginWatcherStop chan struct{}
}

func NewApp() *App {
	a := &App{}
	cfg, home, err := config.LoadConfig()
	if err == nil {
		a.Home = home
		kwCfg := keywords.DefaultConfig()
		if cfg != nil {
			if node, exists := cfg.RawModules["keywords"]; exists {
				_ = node.Decode(&kwCfg)
			}
		}
		llmCfg := a.getLLMConfig(kwCfg.LLMID)
		if cfg != nil {
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
			HomeDir: home,
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
	kwCfg := keywords.DefaultConfig()
	if cfg != nil {
		if node, exists := cfg.RawModules["keywords"]; exists {
			_ = node.Decode(&kwCfg)
		}
	}
	llmCfg := a.getLLMConfig(kwCfg.LLMID)
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
	if home != "" {
		a.startPluginWatcher(filepath.Join(home, ".finnegan", "plugins"))
	}
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
	a.stopPluginWatcher()
	if a.CardManager != nil {
		a.CardManager.Close()
	}
	watcher.GetWatcher().Close()
	bus.GetBus().Close()
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) getLLMConfig(preferredID ...string) module.LLMConfig {
	cfg, _, err := config.LoadConfig()
	if err == nil && cfg != nil {
		// 1. Explicit preferred ID
		if len(preferredID) > 0 && preferredID[0] != "" {
			if llm, ok := cfg.LLMs[preferredID[0]]; ok && llm.APIKey != "" {
				return llm
			}
		}

		// 2. Explicit keywords module config if present in raw config
		if node, exists := cfg.RawModules["keywords"]; exists {
			var kCfg struct {
				LLMID string `yaml:"llm_id"`
			}
			if err := node.Decode(&kCfg); err == nil && kCfg.LLMID != "" {
				if llm, ok := cfg.LLMs[kCfg.LLMID]; ok && llm.APIKey != "" {
					return llm
				}
			}
		}

		// 3. Global explicit default model (e.g. default_llm / default_llm_id)
		if defID := cfg.GetDefaultLLMID(); defID != "" {
			if llm, ok := cfg.LLMs[defID]; ok && llm.APIKey != "" {
				return llm
			}
		}

		// 4. "default" LLM if present and configured
		if def, ok := cfg.LLMs["default"]; ok && def.APIKey != "" {
			return def
		}

		// 4. Smart fallback: find any Gemini provider with a valid API key
		for _, llm := range cfg.LLMs {
			if strings.EqualFold(llm.Provider, "gemini") && llm.APIKey != "" {
				return llm
			}
		}

		// 5. Smart fallback: find ANY configured LLM with a valid API key
		for _, llm := range cfg.LLMs {
			if llm.APIKey != "" {
				return llm
			}
		}

		// 6. If default entry exists (even with empty API key, might be patched later)
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
