package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/chatlog"
	"Finnegan/core/config"
	"Finnegan/core/keywords"
	"Finnegan/core/module"
	"Finnegan/core/search"
	"Finnegan/core/watcher"
	"Finnegan/modules/bugu"
	"Finnegan/modules/bugu/api"
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

func sanitizeSlug(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "_")
	s = strings.ReplaceAll(s, "\t", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.Trim(s, "._ ")
	if len([]rune(s)) > 20 {
		runes := []rune(s)
		return strings.TrimRight(string(runes[:20]), "._ ")
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
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.epiphany.md", baseName, suffix))
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

func (a *App) saveAssetImages(images []ImagePayload) ([]string, error) {
	if len(images) == 0 {
		return nil, nil
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

	assetsDir := filepath.Join(home, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create assets directory: %w", err)
	}

	var savedImageNames []string
	for _, img := range images {
		raw := strings.TrimSpace(img.Data)
		if raw == "" {
			continue
		}

		ext := ""
		if strings.HasPrefix(raw, "data:") {
			idx := strings.Index(raw, ",")
			if idx != -1 {
				header := raw[:idx]
				raw = raw[idx+1:]
				if strings.HasPrefix(header, "data:image/") {
					mimeType := strings.TrimPrefix(header, "data:image/")
					if semi := strings.Index(mimeType, ";"); semi != -1 {
						mimeType = mimeType[:semi]
					}
					switch strings.ToLower(mimeType) {
					case "jpeg", "jpg":
						ext = "jpg"
					case "png":
						ext = "png"
					case "gif":
						ext = "gif"
					case "webp":
						ext = "webp"
					case "svg+xml", "svg":
						ext = "svg"
					case "bmp":
						ext = "bmp"
					case "avif":
						ext = "avif"
					default:
						ext = mimeType
					}
				}
			}
		}

		if img.Ext != "" {
			ext = strings.TrimPrefix(strings.TrimSpace(img.Ext), ".")
		} else if img.Name != "" {
			if e := filepath.Ext(img.Name); e != "" {
				ext = strings.TrimPrefix(e, ".")
			}
		}
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			ext = "png"
		}
		if ext == "jpeg" {
			ext = "jpg"
		}

		cleanB64 := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, raw)

		decoded, err := base64.StdEncoding.DecodeString(cleanB64)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(cleanB64)
			if err != nil {
				decoded, err = base64.URLEncoding.DecodeString(cleanB64)
				if err != nil {
					return nil, fmt.Errorf("failed to decode base64 image: %w", err)
				}
			}
		}

		hashBytes := sha256.Sum256(decoded)
		hashStr := hex.EncodeToString(hashBytes[:])
		fileName := fmt.Sprintf("%d-%s.%s", time.Now().UnixMilli(), hashStr, ext)

		filePath := filepath.Join(assetsDir, fileName)
		if err := os.WriteFile(filePath, decoded, 0644); err != nil {
			return nil, fmt.Errorf("failed to save asset file: %w", err)
		}

		savedImageNames = append(savedImageNames, fileName)
		time.Sleep(time.Millisecond)
	}

	return savedImageNames, nil
}

func (a *App) CaptureThoughtWithImages(text string, images []ImagePayload) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" && len(images) == 0 {
		return "", errors.New("empty text and images")
	}

	savedImageNames, err := a.saveAssetImages(images)
	if err != nil {
		return "", err
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
		return "", errors.New("home directory not configured")
	}

	targetDir := filepath.Join(home, "Scattering")
	now := time.Now()
	slugSrc := text
	if slugSrc == "" {
		slugSrc = "thought"
	}
	baseName := fmt.Sprintf("%s-%s", now.Format("20060102-1504"), sanitizeSlug(slugSrc))
	var targetPath string

	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.epiphany.md", baseName, suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		return "", errors.New("could not find a unique filename")
	}

	docID := ulid.Make().String()

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", docID))
	sb.WriteString("type: thought\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", now.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("source: local\n")
	sb.WriteString("tags: []\n")
	sb.WriteString("publish: false\n")
	if len(savedImageNames) > 0 {
		sb.WriteString("images:\n")
		for _, imgName := range savedImageNames {
			sb.WriteString(fmt.Sprintf("  - %s\n", imgName))
		}
	} else {
		sb.WriteString("images: []\n")
	}
	sb.WriteString("---\n\n")
	sb.WriteString(text)
	sb.WriteString("\n")

	contentStr := sb.String()

	err = watcher.GetWatcher().WriteFile(targetPath, []byte(contentStr), "capture")
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
			Images:     savedImageNames,
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

func (a *App) EvictTempCard(cardID string) bool {
	if a.CardManager == nil {
		return false
	}
	return a.CardManager.EvictTempCard(cardID)
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

	titleRes, _ := a.DeriveChatLogTitle(parsed.Coalesced, parsed.Participants)
	title := titleRes.Title
	if len(titleRes.Participants) > 0 {
		parsed.Participants = titleRes.Participants
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

	titleRes, _ := a.DeriveChatLogTitle(parsed.Coalesced, parsed.Participants)
	var title string
	if titleRes != nil {
		title = titleRes.Title
		if len(titleRes.Participants) > 0 {
			parsed.Participants = titleRes.Participants
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

