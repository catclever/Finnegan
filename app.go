package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/keywords"
	"Finnegan/core/module"
	"Finnegan/core/watcher"
	"Finnegan/modules/bugu"
	"Finnegan/modules/bugu/models"

	"github.com/oklog/ulid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	buguModule    *bugu.BuguModule
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
		}
		a.KeywordEngine = keywords.NewKeywordEngine(llmCfg, kwCfg)
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
				ID:        fmt.Sprintf("%s-radar-%d", docID, i),
				Type:      CardTypeBabel,
				Title:     fmt.Sprintf("Distilled Insights #%d", i+1),
				Content:   res,
				EchoOf:    docID,
				CreatedAt: time.Now(),
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
			var matchedAnchor, echoReason string

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
				matchedAnchor = bc.MatchedAnchor
				echoReason = bc.EchoReason
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
				matchedAnchor = bcp.MatchedAnchor
				echoReason = bcp.EchoReason
			}

			// Filter out empty cards completely: if there is no substantive text, do not inject or emit
			if strings.TrimSpace(content) == "" && strings.TrimSpace(origText) == "" && strings.TrimSpace(transText) == "" {
				continue
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
							MatchedAnchor:  matchedAnchor,
							EchoReason:     echoReason,
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
					"source_url":     srcURL,
					"original_text":  origText,
					"translated_text": transText,
					"publishedAt":    publishedAt,
					"published_at":   publishedAt,
					"authorName":     authorName,
					"author_name":    authorName,
					"authorAvatar":   authorAvatar,
					"author_avatar":  authorAvatar,
					"echoOf":         echoOf,
					"echo_of":        echoOf,
					"matchedAnchor":  matchedAnchor,
					"matched_anchor": matchedAnchor,
					"echoReason":     echoReason,
					"echo_reason":    echoReason,
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

	radarDoneCh := bus.GetBus().Subscribe(bus.EventBuguRadarCompleted, "wails-frontend")
	go func() {
		for event := range radarDoneCh {
			time.Sleep(120 * time.Millisecond)
			runtime.EventsEmit(a.ctx, "bugu:radar_completed", event.Data)
		}
	}()

	// Initialize / refresh KeywordEngine
	llmCfg := a.getLLMConfig()
	kwCfg := keywords.DefaultConfig()
	if cfg != nil {
		if node, exists := cfg.RawModules["keywords"]; exists {
			_ = node.Decode(&kwCfg)
		}
	}
	if a.KeywordEngine == nil {
		a.KeywordEngine = keywords.NewKeywordEngine(llmCfg, kwCfg)
	} else {
		a.KeywordEngine.LLMConfig = llmCfg
		a.KeywordEngine.Config = kwCfg
	}

	kwThoughtCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "keyword-engine")
	go func() {
		for event := range kwThoughtCh {
			if data, ok := event.Data.(bus.EventThoughtCapturedData); ok {
				// Note cards are factual commands/cheatsheets with no emotional anchors; bypass keyword engine.
				if data.Type == "note" {
					continue
				}
				if a.KeywordEngine != nil && data.FilePath != "" && data.DocID != "" {
					a.KeywordEngine.ProcessCard(data.FilePath, data.DocID, data.Text)
				}
			}
		}
	}()

	kwExtractedCh := bus.GetBus().Subscribe(bus.EventKeywordsExtracted, "wails-frontend")
	go func() {
		for event := range kwExtractedCh {
			runtime.EventsEmit(a.ctx, "keywords:extracted", event.Data)
		}
	}()
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
