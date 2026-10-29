package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/keywords"
	"Finnegan/core/module"
	"Finnegan/core/watcher"
	_ "Finnegan/modules/atproto"
	atprotofetcher "Finnegan/modules/atproto/fetcher"
	atprotomodels "Finnegan/modules/atproto/models"
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
	for name, m := range reg {
		mCtx := &module.Context{
			LLMs:    cfg.LLMs,
			Storage: nil,
			Watcher: nil,
		}
		if err := m.Init(mCtx); err != nil {
			log.Printf("[%s] module init failed: %v", name, err)
		} else {
			if bModule, ok := m.(*bugu.BuguModule); ok {
				a.buguModule = bModule
			}
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
			var origin string = "bugu"
			var replyCount, repostCount, likeCount int

			if bc, ok := event.Data.(models.BabelCard); ok {
				if bc.ID != "" {
					cardID = bc.ID
				} else {
					cardID = fmt.Sprintf("babel_%s", ulid.Make().String())
				}
				title = bc.Title
				if title == "" {
					title = "Babel Echo"
				}
				origText = bc.OriginalText
				transText = bc.TranslatedText
				if bc.Content != "" {
					content = bc.Content
				} else if transText != "" && transText != origText {
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
				origin = bc.Origin
				if origin == "" {
					origin = "bugu"
				}
				replyCount = bc.ReplyCount
				repostCount = bc.RepostCount
				likeCount = bc.LikeCount
			} else if bcp, ok := event.Data.(*models.BabelCard); ok && bcp != nil {
				if bcp.ID != "" {
					cardID = bcp.ID
				} else {
					cardID = fmt.Sprintf("babel_%s", ulid.Make().String())
				}
				title = bcp.Title
				if title == "" {
					title = "Babel Echo"
				}
				origText = bcp.OriginalText
				transText = bcp.TranslatedText
				if bcp.Content != "" {
					content = bcp.Content
				} else if transText != "" && transText != origText {
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
				origin = bcp.Origin
				if origin == "" {
					origin = "bugu"
				}
				replyCount = bcp.ReplyCount
				repostCount = bcp.RepostCount
				likeCount = bcp.LikeCount
			} else if post, ok := event.Data.(atprotomodels.Post); ok {
				cardID = fmt.Sprintf("babel_%s", ulid.Make().String())
				title = "Bluesky Echo"
				origText = post.OriginalText
				if post.Content != "" {
					content = post.Content
				} else {
					content = post.OriginalText
				}
				srcURL = post.SourceURL
				echoOf = post.EchoOf
				authorName = post.AuthorName
				authorAvatar = post.AuthorAvatar
				origin = "atproto"
				replyCount = post.ReplyCount
				repostCount = post.RepostCount
				likeCount = post.LikeCount
			} else if postP, ok := event.Data.(*atprotomodels.Post); ok && postP != nil {
				cardID = fmt.Sprintf("babel_%s", ulid.Make().String())
				title = "Bluesky Echo"
				origText = postP.OriginalText
				if postP.Content != "" {
					content = postP.Content
				} else {
					content = postP.OriginalText
				}
				srcURL = postP.SourceURL
				echoOf = postP.EchoOf
				authorName = postP.AuthorName
				authorAvatar = postP.AuthorAvatar
				origin = "atproto"
				replyCount = postP.ReplyCount
				repostCount = postP.RepostCount
				likeCount = postP.LikeCount
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
							Origin:         origin,
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
							ReplyCount:     replyCount,
							RepostCount:    repostCount,
							LikeCount:      likeCount,
							CreatedAt:      time.Now(),
						},
					})
				}
				payload := map[string]interface{}{
					"id":              cardID,
					"type":            "babel",
					"origin":          origin,
					"title":           title,
					"content":         content,
					"sourceUrl":       srcURL,
					"originalText":    origText,
					"translatedText":  transText,
					"source_url":      srcURL,
					"original_text":   origText,
					"translated_text": transText,
					"publishedAt":     publishedAt,
					"published_at":    publishedAt,
					"authorName":      authorName,
					"author_name":     authorName,
					"authorAvatar":    authorAvatar,
					"author_avatar":   authorAvatar,
					"echoOf":          echoOf,
					"echo_of":         echoOf,
					"matchedAnchor":   matchedAnchor,
					"matched_anchor":  matchedAnchor,
					"echoReason":      echoReason,
					"echo_reason":     echoReason,
					"replyCount":      replyCount,
					"repostCount":     repostCount,
					"likeCount":       likeCount,
					"createdAt":       time.Now(),
				}
				runtime.EventsEmit(a.ctx, "babel:card_spawned", payload)
				runtime.EventsEmit(a.ctx, "bugu:card_spawned", payload)
			} else {
				runtime.EventsEmit(a.ctx, "babel:card_spawned", event.Data)
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
	babelDoneCh := bus.GetBus().Subscribe(bus.EventBabelSessionCompleted, "wails-frontend")
	var lastDoneMu sync.Mutex
	var lastDoneID string
	var lastDoneTime time.Time

	handleSessionCompleted := func(data interface{}) {
		var id string
		if m, ok := data.(map[string]string); ok {
			if d, ok := m["cardId"]; ok && d != "" {
				id = d
			} else if d, ok := m["docId"]; ok && d != "" {
				id = d
			}
		}
		lastDoneMu.Lock()
		now := time.Now()
		if id != "" && id == lastDoneID && now.Sub(lastDoneTime) < 500*time.Millisecond {
			lastDoneMu.Unlock()
			return
		}
		lastDoneID = id
		lastDoneTime = now
		lastDoneMu.Unlock()

		time.Sleep(120 * time.Millisecond)
		runtime.EventsEmit(a.ctx, "babel:session_completed", data)
		runtime.EventsEmit(a.ctx, "bugu:radar_completed", data)
	}

	go func() {
		for event := range radarDoneCh {
			handleSessionCompleted(event.Data)
		}
	}()
	go func() {
		for event := range babelDoneCh {
			handleSessionCompleted(event.Data)
		}
	}()

	runtime.EventsOn(a.ctx, "atproto:fetch_comments", func(data ...interface{}) {
		if len(data) == 0 || data[0] == nil {
			return
		}
		var uri, parentCardID string
		if m, ok := data[0].(map[string]interface{}); ok {
			if u, ok := m["uri"].(string); ok {
				uri = u
			}
			if p, ok := m["parentCardId"].(string); ok {
				parentCardID = p
			} else if p, ok := m["parentCardID"].(string); ok {
				parentCardID = p
			}
		}

		if uri == "" {
			return
		}

		f := atprotofetcher.NewFetcher()
		go func() {
			comments, err := f.FetchComments(uri, parentCardID)
			if err != nil {
				log.Printf("[ATProto] FetchComments failed for %s: %v", uri, err)
				return
			}
			for _, comment := range comments {
				bus.GetBus().Publish(bus.Event{
					Type: bus.EventBabelCardSpawned,
					Data: comment,
				})
			}
		}()
	})

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

	fileModCh := bus.GetBus().Subscribe(bus.EventFileModified, "wails-frontend-card-update")
	go func() {
		for event := range fileModCh {
			if data, ok := event.Data.(bus.EventFileData); ok {
				if card, err := ParseCardFile(data.FilePath); err == nil && card != nil {
					if data.DocID != "" {
						card.ID = data.DocID
					}
					runtime.EventsEmit(a.ctx, "card:updated", card)
				}
			}
		}
	}()

	fileDelCh := bus.GetBus().Subscribe(bus.EventFileDeleted, "wails-frontend-card-delete")
	go func() {
		for event := range fileDelCh {
			if data, ok := event.Data.(bus.EventFileData); ok {
				runtime.EventsEmit(a.ctx, "card:deleted", map[string]string{
					"id":       data.DocID,
					"filePath": data.FilePath,
				})
			}
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
