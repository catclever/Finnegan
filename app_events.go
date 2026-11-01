package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"Finnegan/core/babel"
	"Finnegan/core/bus"
	coremodels "Finnegan/core/models"
	"Finnegan/core/module"
	"Finnegan/core/watcher"

	"github.com/oklog/ulid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// appHostAPI implements module.HostAPI, bridging module events to the Finnegan EventBus and Wails runtime.
type appHostAPI struct {
	app *App
}

func (h *appHostAPI) SpawnCard(card coremodels.BabelCard) {
	bus.GetBus().Publish(bus.Event{
		Type: bus.EventBabelCardSpawned,
		Data: card,
	})
}

func (h *appHostAPI) CompleteSession(cardID string) {
	bus.GetBus().Publish(bus.Event{
		Type: bus.EventBabelSessionCompleted,
		Data: map[string]string{"cardId": cardID},
	})
}

func (h *appHostAPI) EmitEvent(eventName string, data interface{}) {
	if h.app != nil && h.app.ctx != nil {
		runtime.EventsEmit(h.app.ctx, eventName, data)
	}
}

// normalizeBabelCardData converts a coremodels.BabelCard, legacy module struct (e.g. atproto Post),
// or map[string]interface{} into a normalized coremodels.BabelCard without compile-time module coupling.
func normalizeBabelCardData(raw interface{}) (coremodels.BabelCard, bool) {
	if raw == nil {
		return coremodels.BabelCard{}, false
	}
	if bc, ok := raw.(coremodels.BabelCard); ok {
		return bc, true
	}
	if bcp, ok := raw.(*coremodels.BabelCard); ok && bcp != nil {
		return *bcp, true
	}

	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return coremodels.BabelCard{}, false
	}

	var flex struct {
		ID             string `json:"id"`
		Origin         string `json:"origin"`
		Title          string `json:"title"`
		Content        string `json:"content"`
		SourceURL      string `json:"source_url"`
		SourceURLAlt   string `json:"sourceUrl"`
		OriginalText   string `json:"original_text"`
		OriginalAlt    string `json:"originalText"`
		TranslatedText string `json:"translated_text"`
		TranslatedAlt  string `json:"translatedText"`
		PublishedAt    string `json:"published_at"`
		PublishedAlt   string `json:"publishedAt"`
		AuthorName     string `json:"author_name"`
		AuthorNameAlt  string `json:"authorName"`
		AuthorAvatar   string `json:"author_avatar"`
		AuthorAvAlt    string `json:"authorAvatar"`
		EchoOf         string `json:"echo_of"`
		EchoOfAlt      string `json:"echoOf"`
		MatchedAnchor  string `json:"matched_anchor"`
		MatchedAlt     string `json:"matchedAnchor"`
		EchoReason     string `json:"echo_reason"`
		EchoReasonAlt  string `json:"echoReason"`
		ReplyCount     int    `json:"reply_count"`
		ReplyCountAlt  int    `json:"replyCount"`
		RepostCount    int    `json:"repost_count"`
		RepostCountAlt int    `json:"repostCount"`
		LikeCount      int    `json:"like_count"`
		LikeCountAlt   int    `json:"likeCount"`
	}
	if err := json.Unmarshal(rawBytes, &flex); err != nil {
		return coremodels.BabelCard{}, false
	}

	pickStr := func(a, b string) string {
		if a != "" {
			return a
		}
		return b
	}
	pickInt := func(a, b int) int {
		if a != 0 {
			return a
		}
		return b
	}

	origin := flex.Origin
	title := flex.Title

	// Detect legacy atproto Post structs via reflection type name or package path
	rt := reflect.TypeOf(raw)
	if rt != nil && rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt != nil && (rt.Name() == "Post" || strings.Contains(rt.PkgPath(), "atproto")) {
		if origin == "" {
			origin = "atproto"
		}
		if title == "" {
			title = "Bluesky Echo"
		}
	}

	return coremodels.BabelCard{
		ID:             flex.ID,
		Origin:         origin,
		Title:          title,
		Content:        flex.Content,
		SourceURL:      pickStr(flex.SourceURL, flex.SourceURLAlt),
		OriginalText:   pickStr(flex.OriginalText, flex.OriginalAlt),
		TranslatedText: pickStr(flex.TranslatedText, flex.TranslatedAlt),
		PublishedAt:    pickStr(flex.PublishedAt, flex.PublishedAlt),
		AuthorName:     pickStr(flex.AuthorName, flex.AuthorNameAlt),
		AuthorAvatar:   pickStr(flex.AuthorAvatar, flex.AuthorAvAlt),
		EchoOf:         pickStr(flex.EchoOf, flex.EchoOfAlt),
		MatchedAnchor:  pickStr(flex.MatchedAnchor, flex.MatchedAlt),
		EchoReason:     pickStr(flex.EchoReason, flex.EchoReasonAlt),
		ReplyCount:     pickInt(flex.ReplyCount, flex.ReplyCountAlt),
		RepostCount:    pickInt(flex.RepostCount, flex.RepostCountAlt),
		LikeCount:      pickInt(flex.LikeCount, flex.LikeCountAlt),
	}, true
}

// setupWatcherCallbacks configures reactive event callbacks for filesystem changes, deletions, and conflicts.
func (a *App) setupWatcherCallbacks(globalWatcher *watcher.Watcher, watcherStorage *WatcherStorage) {
	if watcherStorage == nil {
		return
	}
	watcherStorage.SetOnChange(func(id, data string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "notes:changed", map[string]interface{}{
				"docId": id,
				"op":    "upsert",
			})
		}
	})
	watcherStorage.SetOnDelete(func(id string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "notes:changed", map[string]interface{}{
				"docId": id,
				"op":    "remove",
			})
		}
	})
	watcherStorage.SetOnConflict(func(id string, paths []string) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "card:id_conflict", map[string]interface{}{
				"id":    id,
				"paths": paths,
			})
		}
	})
}

// onThoughtCapturedEvent handles thought capture events by broadcasting them to the frontend runtime.
func (a *App) onThoughtCapturedEvent(msg bus.Message) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "thought:captured", msg.Data)
	}
}

// onBabelCardSpawned handles external/distilled card spawning from Bugu, ATProto, or Babel streams.
func (a *App) onBabelCardSpawned(event bus.Message) {
	var cardID string
	var title, content, origText, transText, srcURL, echoOf string
	var publishedAt, authorName, authorAvatar string
	var matchedAnchor, echoReason string
	var origin string = "bugu"
	var replyCount, repostCount, likeCount int

	if bc, ok := normalizeBabelCardData(event.Data); ok {
		if bc.ID != "" {
			cardID = bc.ID
		} else {
			cardID = fmt.Sprintf("babel_%s", ulid.Make().String())
		}
		origin = bc.Origin
		if origin == "" {
			origin = "bugu"
		}
		title = bc.Title
		if title == "" {
			if origin == "atproto" {
				title = "Bluesky Echo"
			} else {
				title = "Babel Echo"
			}
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
		replyCount = bc.ReplyCount
		repostCount = bc.RepostCount
		likeCount = bc.LikeCount
	}

	// Filter out empty cards completely: if there is no substantive text, do not inject or emit
	if strings.TrimSpace(content) == "" && strings.TrimSpace(origText) == "" && strings.TrimSpace(transText) == "" {
		return
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
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "babel:card_spawned", payload)
			runtime.EventsEmit(a.ctx, "bugu:card_spawned", payload)
		}
	} else if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "babel:card_spawned", event.Data)
		runtime.EventsEmit(a.ctx, "bugu:card_spawned", event.Data)
	}
}

// handleSessionCompleted handles debouncing and broadcasting radar and babel session completion events.
func (a *App) handleSessionCompleted(data interface{}, lastDoneMu *sync.Mutex, lastDoneID *string, lastDoneTime *time.Time) {
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
	if id != "" && id == *lastDoneID && now.Sub(*lastDoneTime) < 500*time.Millisecond {
		lastDoneMu.Unlock()
		return
	}
	*lastDoneID = id
	*lastDoneTime = now
	lastDoneMu.Unlock()

	time.Sleep(120 * time.Millisecond)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "babel:session_completed", data)
		runtime.EventsEmit(a.ctx, "bugu:radar_completed", data)
	}
}

// handleAtprotoFetchComments fetches ATProto comment threads asynchronously and publishes spawned cards to the event bus.
func (a *App) handleAtprotoFetchComments(data ...interface{}) {
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

	if m, ok := module.GetRegistry()["atproto"]; ok {
		if ap, ok := m.(module.ActionProvider); ok {
			go func() {
				ctx := a.ctx
				if ctx == nil {
					ctx = context.Background()
				}
				if _, err := ap.ExecuteAction(ctx, "fetch_comments", map[string]interface{}{
					"uri":          uri,
					"parentCardId": parentCardID,
				}); err != nil {
					log.Printf("[ATProto] ExecuteAction(fetch_comments) failed for %s: %v", uri, err)
				}
			}()
			return
		}
	}

	go func() {
		comments, err := babel.FetchBskyPostComments(uri, parentCardID)
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
}

// onKeywordEngineThoughtCaptured routes non-note thought cards through keyword extraction.
func (a *App) onKeywordEngineThoughtCaptured(msg bus.Message) {
	if data, ok := msg.Data.(bus.EventThoughtCapturedData); ok {
		// Note cards are factual commands/cheatsheets with no emotional anchors; bypass keyword engine.
		if data.Type == "note" {
			return
		}
		if a.KeywordEngine != nil && data.FilePath != "" && data.DocID != "" {
			a.KeywordEngine.ProcessCard(data.FilePath, data.DocID, data.Text)
		}
	}
}

// onFileModified updates card storage and emits card:updated when a file is modified externally.
func (a *App) onFileModified(msg bus.Message) {
	if data, ok := msg.Data.(bus.EventFileData); ok {
		if card, err := ParseCardFile(data.FilePath); err == nil && card != nil {
			if data.DocID != "" {
				card.ID = data.DocID
			}
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "card:updated", card)
			}
		}
	}
}

// onFileDeleted emits card:deleted when a card file is removed externally.
func (a *App) onFileDeleted(msg bus.Message) {
	if data, ok := msg.Data.(bus.EventFileData); ok {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "card:deleted", map[string]string{
				"id":       data.DocID,
				"filePath": data.FilePath,
			})
		}
	}
}

// setupEventSubscriptions registers all event bus subscriptions and runtime listeners.
func (a *App) setupEventSubscriptions() {
	babelCh := bus.GetBus().Subscribe(bus.EventBabelCardSpawned, "wails-frontend")
	go func() {
		for event := range babelCh {
			a.onBabelCardSpawned(event)
		}
	}()

	thoughtCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "wails-frontend")
	go func() {
		for event := range thoughtCh {
			a.onThoughtCapturedEvent(event)
		}
	}()

	radarDoneCh := bus.GetBus().Subscribe(bus.EventBuguRadarCompleted, "wails-frontend")
	babelDoneCh := bus.GetBus().Subscribe(bus.EventBabelSessionCompleted, "wails-frontend")
	var lastDoneMu sync.Mutex
	var lastDoneID string
	var lastDoneTime time.Time

	go func() {
		for event := range radarDoneCh {
			a.handleSessionCompleted(event.Data, &lastDoneMu, &lastDoneID, &lastDoneTime)
		}
	}()
	go func() {
		for event := range babelDoneCh {
			a.handleSessionCompleted(event.Data, &lastDoneMu, &lastDoneID, &lastDoneTime)
		}
	}()

	if a.ctx != nil {
		runtime.EventsOn(a.ctx, "atproto:fetch_comments", a.handleAtprotoFetchComments)
	}

	kwThoughtCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "keyword-engine")
	go func() {
		for event := range kwThoughtCh {
			a.onKeywordEngineThoughtCaptured(event)
		}
	}()

	kwExtractedCh := bus.GetBus().Subscribe(bus.EventKeywordsExtracted, "wails-frontend")
	go func() {
		for event := range kwExtractedCh {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "keywords:extracted", event.Data)
			}
		}
	}()

	fileModCh := bus.GetBus().Subscribe(bus.EventFileModified, "wails-frontend-card-update")
	go func() {
		for event := range fileModCh {
			a.onFileModified(event)
		}
	}()

	fileDelCh := bus.GetBus().Subscribe(bus.EventFileDeleted, "wails-frontend-card-delete")
	go func() {
		for event := range fileDelCh {
			a.onFileDeleted(event)
		}
	}()
}
