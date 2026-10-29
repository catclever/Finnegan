package atproto

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"Finnegan/core/bus"
	"Finnegan/core/module"
	"Finnegan/modules/atproto/config"
	"Finnegan/modules/atproto/fetcher"
	"Finnegan/modules/atproto/publisher"
)

type AtProtoModule struct {
	cfg       *config.Config
	publisher *publisher.Publisher
	fetcher   *fetcher.Fetcher
	ctx       *module.Context
	API       *AtProtoAPI
}

var _ module.Module = (*AtProtoModule)(nil)
var _ module.CommandProvider = (*AtProtoModule)(nil)

// AtProtoAPI defines the methods exposed to the Wails frontend (Svelte)
type AtProtoAPI struct {
	publisher *publisher.Publisher
	fetcher   *fetcher.Fetcher
}

func init() {
	module.Register(&AtProtoModule{
		cfg: &config.Config{
			PDSUrl: "https://bsky.social",
		},
	})
}

func (m *AtProtoModule) Name() string {
	return "atproto"
}

func (m *AtProtoModule) ConfigTarget() interface{} {
	return m.cfg
}

func (m *AtProtoModule) Init(ctx *module.Context) error {
	m.ctx = ctx
	m.publisher = publisher.NewPublisher(m.cfg)
	m.fetcher = fetcher.NewFetcher()

	m.API = &AtProtoAPI{
		publisher: m.publisher,
		fetcher:   m.fetcher,
	}

	log.Printf("[ATProto] Module ready. Manual trigger mode enabled.")
	return nil
}

// Bindings exposes the API struct to the Wails frontend
func (m *AtProtoModule) Bindings() []interface{} {
	return []interface{}{
		m.API,
	}
}

// GetCommands returns the Omnibar commands provided by AtProtoModule
func (m *AtProtoModule) GetCommands() []module.CommandDef {
	return []module.CommandDef{
		{
			ID:          "atproto:echo",
			Name:        "Bluesky 灵感回声",
			Trigger:     "bsky",
			Description: "在 Bluesky (ATProto) 寻找当前卡片或关键词的共鸣与讨论",
			Namespace:   "bsky",
			IsPrimary:   true,
			Params: []module.ParamSpec{
				{
					Name:                "card",
					Type:                module.ParamTypeCard,
					Description:         "选择卡片或输入搜索词",
					DataSource:          "cards",
					RequireKeywords:     true,
					ExcludedTypes:       []string{"note"},
					AutoStageToViewport: true,
				},
			},
		},
		{
			ID:          "atproto:publish",
			Name:        "发布到 Bluesky",
			Trigger:     "bsky-push",
			Description: "将卡片发布至 Bluesky (手动触发)",
			Namespace:   "bsky",
			IsPrimary:   false,
			SubCommand:  "push",
			Params: []module.ParamSpec{
				{
					Name:                "card",
					Type:                module.ParamTypeCard,
					Description:         "选择要发布的卡片",
					DataSource:          "cards",
					RequireKeywords:     false,
					ExcludedTypes:       []string{"babel"},
					AutoStageToViewport: true,
				},
			},
		},
	}
}

// extractKeywordsFromMap collects non-empty keywords from card metadata keywords map
func extractKeywordsFromMap(kwObj interface{}) []string {
	var results []string
	seen := make(map[string]bool)

	addWord := func(w string) {
		w = strings.TrimSpace(w)
		w = strings.TrimSuffix(w, "*")
		if w != "" && !seen[w] {
			seen[w] = true
			results = append(results, w)
		}
	}

	switch m := kwObj.(type) {
	case map[string]interface{}:
		for _, v := range m {
			if list, ok := v.([]interface{}); ok {
				for _, item := range list {
					if s, ok := item.(string); ok {
						addWord(s)
					}
				}
			} else if list, ok := v.([]string); ok {
				for _, s := range list {
					addWord(s)
				}
			}
		}
	case map[string][]string:
		for _, list := range m {
			for _, s := range list {
				addWord(s)
			}
		}
	}
	return results
}

// tokenizeText splits text into keyword tokens when card metadata has no keywords
func tokenizeText(text string) []string {
	clean := strings.Map(func(r rune) rune {
		if r == '#' || r == '*' || r == '`' || r == '>' || r == '[' || r == ']' || r == '(' || r == ')' {
			return ' '
		}
		return r
	}, text)

	fields := strings.FieldsFunc(clean, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == ',' || r == '.' || r == '!' || r == '?' ||
			r == ';' || r == ':' || r == '，' || r == '。' || r == '！' || r == '？' || r == '；' || r == '：' ||
			r == '（' || r == '）' || r == '【' || r == '】' || r == '"' || r == '\'' || r == '“' || r == '”'
	})

	var keywords []string
	seen := make(map[string]bool)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if len([]rune(f)) >= 2 && !seen[f] {
			seen[f] = true
			keywords = append(keywords, f)
			if len(keywords) >= 8 {
				break
			}
		}
	}
	return keywords
}

// ExecuteCommand executes an Omnibar command implemented by AtProtoModule
func (m *AtProtoModule) ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error) {
	switch cmdID {
	case "atproto:echo":
		var content string
		var cardID string
		var keywords []string

		if c, ok := params["content"].(string); ok && c != "" {
			content = c
		} else if text, ok := params["text"].(string); ok && text != "" {
			content = text
		}

		if id, ok := params["id"].(string); ok && id != "" {
			cardID = id
		} else if cid, ok := params["cardId"].(string); ok && cid != "" {
			cardID = cid
		}

		if cardVal, ok := params["card"]; ok {
			switch cv := cardVal.(type) {
			case string:
				if content == "" {
					content = cv
				}
			case map[string]interface{}:
				if content == "" {
					if c, ok := cv["content"].(string); ok && c != "" {
						content = c
					} else if t, ok := cv["title"].(string); ok && t != "" {
						content = t
					}
				}
				if cardID == "" {
					if id, ok := cv["id"].(string); ok && id != "" {
						cardID = id
					}
				}
				if kwObj, ok := cv["keywords"]; ok && kwObj != nil {
					keywords = extractKeywordsFromMap(kwObj)
				}
			}
		}

		if content == "" && len(keywords) == 0 {
			return nil, fmt.Errorf("no content or card provided for %s", cmdID)
		}

		if len(keywords) == 0 {
			keywords = tokenizeText(content)
		}

		if len(keywords) == 0 && content != "" {
			keywords = []string{content}
		}

		// Asynchronously call m.fetcher.SearchSimilar
		go func(kws []string, cID string) {
			f := m.fetcher
			if f == nil {
				f = fetcher.NewFetcher()
			}
			posts, err := f.SearchSimilar(kws, 10, cID)
			if err != nil {
				log.Printf("[ATProto] SearchSimilar failed: %v", err)
			} else {
				b := bus.GetBus()
				for _, post := range posts {
					b.Publish(bus.Event{
						Type: bus.EventBabelCardSpawned,
						Data: post,
					})
				}
			}
			bus.GetBus().Publish(bus.Event{
				Type: bus.EventBuguRadarCompleted,
				Data: map[string]string{"cardId": cID},
			})
		}(keywords, cardID)

		return map[string]interface{}{
			"status":   "triggered",
			"command":  cmdID,
			"target":   content,
			"card_id":  cardID,
			"keywords": keywords,
		}, nil

	case "atproto:publish", "atproto:bsky-push":
		var filePath string
		var content string

		if fp, ok := params["filePath"].(string); ok && fp != "" {
			filePath = fp
		}
		if c, ok := params["content"].(string); ok && c != "" {
			content = c
		} else if text, ok := params["text"].(string); ok && text != "" {
			content = text
		}

		if cardVal, ok := params["card"]; ok {
			if cv, ok := cardVal.(map[string]interface{}); ok {
				if filePath == "" {
					if fp, ok := cv["filePath"].(string); ok && fp != "" {
						filePath = fp
					}
				}
				if content == "" {
					if c, ok := cv["content"].(string); ok && c != "" {
						content = c
					} else if t, ok := cv["title"].(string); ok && t != "" {
						content = t
					}
				}
			}
		}

		if content == "" && filePath != "" {
			if data, err := os.ReadFile(filePath); err == nil {
				content = string(data)
			}
		}

		if content == "" {
			return nil, fmt.Errorf("no content provided for %s", cmdID)
		}

		pub := m.publisher
		if pub == nil {
			pub = publisher.NewPublisher(m.cfg)
		}

		uri, cid, err := pub.PublishNote(ctx, filePath, content)
		if err != nil {
			return nil, fmt.Errorf("publish note failed: %w", err)
		}

		return map[string]interface{}{
			"status": "success",
			"uri":    uri,
			"cid":    cid,
		}, nil
	}

	return nil, fmt.Errorf("unknown command: %s", cmdID)
}

// ----------------------------------------------------------------------
// Wails Frontend API Methods (These are converted to JS Promises in Svelte)
// ----------------------------------------------------------------------

// PublishNote allows the user to manually trigger publishing a card to Bluesky from the GUI
func (api *AtProtoAPI) PublishNote(filePath string, content string) (string, error) {
	log.Printf("[ATProto API] User manually triggered publish for: %s", filePath)
	uri, _, err := api.publisher.PublishNote(context.Background(), filePath, content)
	return uri, err
}

// SearchSimilar allows the user to manually trigger a deep dive search on Bluesky
func (api *AtProtoAPI) SearchSimilar(query string, limit int) (string, error) {
	log.Printf("[ATProto API] User triggered search for: %s", query)
	posts, err := api.fetcher.SearchSimilar([]string{query}, limit, "")
	if err != nil {
		return "", err
	}
	b := bus.GetBus()
	for _, post := range posts {
		b.Publish(bus.Event{
			Type: bus.EventBabelCardSpawned,
			Data: post,
		})
	}
	return "Search dispatched", nil
}
