package atproto

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"Finnegan/core/babel"
	"Finnegan/core/bus"
	"Finnegan/core/module"
	"Finnegan/modules/atproto/config"
	"Finnegan/modules/atproto/fetcher"
	"Finnegan/modules/atproto/publisher"
	bugullm "Finnegan/modules/bugu/llm"

	"github.com/oklog/ulid/v2"
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
			Description: "Bluesky 灵感回声",
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
			Description: "发布到 Bluesky",
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

// extractKeywordsFromCard collects non-empty keywords and tags from card metadata
func extractKeywordsFromCard(cardMap map[string]interface{}) []string {
	var results []string
	seen := make(map[string]bool)

	addWord := func(w string, isTag bool) {
		w = strings.TrimSpace(w)
		if isTag {
			w = strings.TrimLeft(w, "#")
		}
		w = strings.TrimRight(w, "*")
		w = strings.TrimSpace(w)
		if w != "" && !seen[w] {
			seen[w] = true
			results = append(results, w)
		}
	}

	processList := func(val interface{}, isTag bool) {
		switch list := val.(type) {
		case []interface{}:
			for _, item := range list {
				if s, ok := item.(string); ok {
					addWord(s, isTag)
				}
			}
		case []string:
			for _, s := range list {
				addWord(s, isTag)
			}
		case string:
			addWord(list, isTag)
		}
	}

	// 1. From cardMap["keywords"]: traverse all dimensions, strip trailing * and whitespace
	if kwObj, ok := cardMap["keywords"]; ok && kwObj != nil {
		switch m := kwObj.(type) {
		case map[string]interface{}:
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				processList(m[k], false)
			}
		case map[string][]string:
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				for _, s := range m[k] {
					addWord(s, false)
				}
			}
		case map[interface{}]interface{}:
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, fmt.Sprintf("%v", k))
			}
			sort.Strings(keys)
			for _, k := range keys {
				for rk, rv := range m {
					if fmt.Sprintf("%v", rk) == k {
						processList(rv, false)
						break
					}
				}
			}
		}
	}

	// 2. From cardMap["tags"]: traverse tag list, strip leading # and trailing *
	if tagObj, ok := cardMap["tags"]; ok && tagObj != nil {
		processList(tagObj, true)
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

		var cardMap map[string]interface{}
		if cardVal, ok := params["card"]; ok && cardVal != nil {
			if cm, ok := cardVal.(map[string]interface{}); ok {
				cardMap = cm
			}
		}

		if cardMap != nil {
			if id, ok := cardMap["id"].(string); ok && id != "" {
				cardID = id
			} else if id, ok := params["id"].(string); ok && id != "" {
				cardID = id
			} else if cid, ok := params["cardId"].(string); ok && cid != "" {
				cardID = cid
			}

			if c, ok := cardMap["content"].(string); ok && c != "" {
				content = c
			} else if t, ok := cardMap["title"].(string); ok && t != "" {
				content = t
			} else if c, ok := params["content"].(string); ok && c != "" {
				content = c
			} else if text, ok := params["text"].(string); ok && text != "" {
				content = text
			}

			keywords = extractKeywordsFromCard(cardMap)
			if len(keywords) == 0 && content != "" {
				keywords = tokenizeText(content)
			}
			if len(keywords) == 0 && content != "" {
				keywords = []string{content}
			}
		} else {
			// Free search mode (NO card provided)
			var rawText string
			if text, ok := params["text"].(string); ok && strings.TrimSpace(text) != "" {
				rawText = text
			} else if c, ok := params["content"].(string); ok && strings.TrimSpace(c) != "" {
				rawText = c
			} else if cardStr, ok := params["card"].(string); ok && strings.TrimSpace(cardStr) != "" {
				rawText = cardStr
			}

			if strings.TrimSpace(rawText) != "" {
				rawText = strings.TrimSpace(rawText)
				keywords = []string{rawText}
				cardID = ""
				content = rawText
			}
		}

		if len(keywords) == 0 {
			return nil, fmt.Errorf("no content or card provided for %s", cmdID)
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
				var llmCfg module.LLMConfig
				if m.ctx != nil && m.ctx.LLMs != nil {
					if c, ok := m.ctx.LLMs["distiller"]; ok {
						llmCfg = c
					} else if c, ok := m.ctx.LLMs["default"]; ok {
						llmCfg = c
					}
				}
				translator := babel.NewUniversalTranslator(llmCfg, "gemini-3.8-flash", bugullm.GenerateCompletion)
				gatekeeper := babel.NewBlueskyGatekeeper(translator)

				b := bus.GetBus()
				for _, post := range posts {
					card, err := gatekeeper.ValidateAndCurate(ctx, babel.RawBabelInput{
						ID:             fmt.Sprintf("babel_%s", ulid.Make().String()),
						Origin:         "atproto",
						URL:            post.SourceURL,
						Title:          "Bluesky Echo",
						OriginalText:   post.OriginalText,
						AuthorName:     post.AuthorName,
						AuthorAvatar:   post.AuthorAvatar,
						EchoOf:         post.EchoOf,
						SearchKeywords: kws,
						ReplyCount:     post.ReplyCount,
						RepostCount:    post.RepostCount,
						LikeCount:      post.LikeCount,
					})
					if err != nil {
						log.Printf("[ATProto Gatekeeper] Discarded post from @%s: %v", post.AuthorName, err)
						continue
					}
					if card == nil {
						continue
					}

					b.Publish(bus.Event{
						Type: bus.EventBabelCardSpawned,
						Data: *card,
					})
				}
			}
			bus.GetBus().Publish(bus.Event{
				Type: bus.EventBabelSessionCompleted,
				Data: map[string]string{"cardId": cID},
			})
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
