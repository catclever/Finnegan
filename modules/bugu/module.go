package bugu

import (
	"context"
	"fmt"
	"log"
	"os"

	"Finnegan/core/module"
	"Finnegan/modules/bugu/api"
	"Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/llm"
	"Finnegan/modules/bugu/models"

	"gopkg.in/yaml.v3"
)

type BuguConfig struct {
	SearxngURL     string `yaml:"searxng_url"`
	MaxConcurrency int    `yaml:"max_concurrency"`
	FallbackKey    string `yaml:"fallback_key"`
	APIKey         string `yaml:"api_key"`
	JinaAPIKey     string `yaml:"jina_api_key"`
	TargetLanguage string `yaml:"target_language"`
	LLMID          string `yaml:"llm_id"`
	RouterLLMID    string `yaml:"router_llm_id"`
	DistillerLLMID string `yaml:"distiller_llm_id"`
	DefaultModel   string `yaml:"default_model"`
	RouterModel    string `yaml:"router_model"`
	DistillerModel string `yaml:"distiller_model"`
}

type BuguModule struct {
	MyConfig     *BuguConfig
	Router       *llm.Router
	RadarTargets *config.Config
}

// Compile-time check that BuguModule implements module.CommandProvider
var _ module.CommandProvider = (*BuguModule)(nil)

func (b *BuguModule) Name() string {
	return "bugu"
}

func (b *BuguModule) ConfigTarget() interface{} {
	if b.MyConfig == nil {
		// Tier 4: Hardcoded Defaults
		b.MyConfig = &BuguConfig{
			SearxngURL:     "http://127.0.0.1:8080",
			MaxConcurrency: 10,
			TargetLanguage: "English",
			LLMID:          "default",
		}

		// Tier 3: Environment Variable
		if envURL := os.Getenv("SEARXNG_URL"); envURL != "" {
			b.MyConfig.SearxngURL = envURL
		}
		if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
			b.MyConfig.FallbackKey = envKey
		}
		if envJina := os.Getenv("JINA_API_KEY"); envJina != "" {
			b.MyConfig.JinaAPIKey = envJina
		}

		// Tier 2: Module Local Config
		data, err := os.ReadFile("bugu.yml")
		if err == nil {
			_ = yaml.Unmarshal(data, b.MyConfig)
		}
	}
	return b.MyConfig
}

func (b *BuguModule) Init(ctx *module.Context) error {
	paths := []string{
		"modules/bugu/radar_targets.yml",
		"bugu/radar_targets.yml",
		"radar_targets.yml",
	}
	var cfg *config.Config
	var err error
	for _, p := range paths {
		cfg, err = config.LoadConfig(p)
		if err == nil && len(cfg.Targets) > 0 {
			break
		}
	}
	if cfg == nil || len(cfg.Targets) == 0 {
		log.Printf("[Bugu Module] Warning: No radar targets loaded: %v", err)
		cfg = &config.Config{Targets: []models.Target{}, TargetMap: make(map[string]models.Target)}
	} else {
		log.Printf("[Bugu Module] Successfully loaded %d radar targets", len(cfg.Targets))
	}
	b.RadarTargets = cfg

	if b.MyConfig == nil {
		b.ConfigTarget()
	}

	// Set models on cfg if specifically configured
	if b.MyConfig.RouterModel != "" {
		cfg.RouterModel = b.MyConfig.RouterModel
	}
	if b.MyConfig.DistillerModel != "" {
		cfg.DistillerModel = b.MyConfig.DistillerModel
	}

	// 1. Resolve Router LLM (router_llm_id -> llm_id -> "default")
	routerID := b.MyConfig.RouterLLMID
	if routerID == "" {
		routerID = b.MyConfig.LLMID
	}
	if routerID == "" {
		routerID = "default"
	}
	rawRouterLLM := ctx.LLMs[routerID]
	routerLLM := module.LLMConfig{
		Provider:     rawRouterLLM.Provider,
		APIKey:       rawRouterLLM.APIKey,
		BaseURL:      rawRouterLLM.BaseURL,
		DefaultModel: rawRouterLLM.DefaultModel,
	}
	if routerLLM.APIKey == "" {
		if b.MyConfig.FallbackKey != "" {
			routerLLM.APIKey = b.MyConfig.FallbackKey
		} else if b.MyConfig.APIKey != "" {
			routerLLM.APIKey = b.MyConfig.APIKey
		} else if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
			routerLLM.APIKey = envKey
		}
	}
	if b.MyConfig.RouterModel != "" {
		routerLLM.DefaultModel = b.MyConfig.RouterModel
	} else if routerLLM.DefaultModel == "" {
		if b.MyConfig.DefaultModel != "" {
			routerLLM.DefaultModel = b.MyConfig.DefaultModel
		} else {
			routerLLM.DefaultModel = "gemini-3.8-flash"
		}
	}

	// 2. Resolve Distiller LLM (distiller_llm_id -> llm_id -> "default")
	distillerID := b.MyConfig.DistillerLLMID
	if distillerID == "" {
		distillerID = b.MyConfig.LLMID
	}
	if distillerID == "" {
		distillerID = "default"
	}
	rawDistillerLLM := ctx.LLMs[distillerID]
	distillerLLM := module.LLMConfig{
		Provider:     rawDistillerLLM.Provider,
		APIKey:       rawDistillerLLM.APIKey,
		BaseURL:      rawDistillerLLM.BaseURL,
		DefaultModel: rawDistillerLLM.DefaultModel,
	}
	if distillerLLM.APIKey == "" {
		if b.MyConfig.FallbackKey != "" {
			distillerLLM.APIKey = b.MyConfig.FallbackKey
		} else if b.MyConfig.APIKey != "" {
			distillerLLM.APIKey = b.MyConfig.APIKey
		} else if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
			distillerLLM.APIKey = envKey
		}
	}
	if b.MyConfig.DistillerModel != "" {
		distillerLLM.DefaultModel = b.MyConfig.DistillerModel
	} else if distillerLLM.DefaultModel == "" {
		if b.MyConfig.DefaultModel != "" {
			distillerLLM.DefaultModel = b.MyConfig.DefaultModel
		} else {
			distillerLLM.DefaultModel = "gemini-3.8-flash"
		}
	}

	log.Printf("[Bugu Module] Router LLM initialized (id: %s, provider: %s, model: %s, base_url: %s)",
		routerID, routerLLM.Provider, routerLLM.DefaultModel, routerLLM.BaseURL)
	log.Printf("[Bugu Module] Distiller LLM initialized (id: %s, provider: %s, model: %s, base_url: %s)",
		distillerID, distillerLLM.Provider, distillerLLM.DefaultModel, distillerLLM.BaseURL)

	// Router initialization using final routerLLM and distillerLLM configs
	b.Router = llm.NewRouter(cfg, routerLLM, distillerLLM)

	// Automatic background enrichment of new/untranslated tags
	if routerLLM.APIKey != "" || routerLLM.BaseURL != "" {
		missingTags := cfg.FindUntranslatedTags()
		if len(missingTags) > 0 {
			log.Printf("[Bugu Module] Detected %d new untranslated tags. Starting background auto-enrichment...", len(missingTags))
			go func() {
				if err := cfg.AutoEnrich(b.Router); err != nil {
					log.Printf("[Bugu Module] Tag auto-enrichment warning: %v", err)
				} else {
					log.Printf("[Bugu Module] Successfully auto-enriched and cached %d tags into %s", len(missingTags), cfg.I18nFilePath)
				}
			}()
		}
	}

	return nil
}

func (b *BuguModule) Bindings() []interface{} {
	return []interface{}{
		b,
	}
}

// GetCommands returns the Omnibar commands provided by BuguModule
func (b *BuguModule) GetCommands() []module.CommandDef {
	return []module.CommandDef{
		{
			ID:          "bugu:radar",
			Name:        "布谷深网雷达",
			Trigger:     "bugu",
			Description: "吾道不孤",
			Namespace:   "bugu",
			IsPrimary:   true,
			Params: []module.ParamSpec{
				{
					Name:                "card",
					Type:                module.ParamTypeCard,
					Description:         "吾道不孤",
					DataSource:          "cards",
					RequireKeywords:     true,
					ExcludedTypes:       []string{"note"},
					AutoStageToViewport: true,
				},
			},
		},
	}
}

// ExecuteCommand executes an Omnibar command implemented by BuguModule
func (b *BuguModule) ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error) {
	if cmdID == "bugu:radar" {
		var content string
		if c, ok := params["content"].(string); ok && c != "" {
			content = c
		} else if text, ok := params["text"].(string); ok && text != "" {
			content = text
		} else if id, ok := params["id"].(string); ok && id != "" {
			content = id
		} else if cardVal, ok := params["card"]; ok {
			switch cv := cardVal.(type) {
			case string:
				content = cv
			case map[string]interface{}:
				if c, ok := cv["content"].(string); ok && c != "" {
					content = c
				} else if t, ok := cv["title"].(string); ok && t != "" {
					content = t
				} else if id, ok := cv["id"].(string); ok && id != "" {
					content = id
				}
			}
		}

		var cardID string
		if id, ok := params["id"].(string); ok && id != "" {
			cardID = id
		}

		if content == "" {
			return nil, fmt.Errorf("no content or card provided for %s", cmdID)
		}

		// Run Radar asynchronously associated with cardID
		go b.RunRadarForDoc(cardID, content)

		return map[string]interface{}{
			"status":  "triggered",
			"command": cmdID,
			"target":  content,
			"card_id": cardID,
		}, nil
	}

	return nil, fmt.Errorf("unknown command: %s", cmdID)
}

// Wails exposed methods

func (b *BuguModule) GetRadarTargets() []models.Target {
	if b.RadarTargets == nil {
		return []models.Target{}
	}
	return b.RadarTargets.Targets
}

func (b *BuguModule) RunRadarForDoc(docID string, intent string) api.RadarResponse {
	if b.RadarTargets == nil || b.Router == nil {
		return api.RadarResponse{
			Status: "error",
			Error:  "Bugu configuration is not loaded",
		}
	}
	var kw map[string][]string
	if docID != "" {
		kw = api.ExtractKeywordsForDoc(docID)
	}
	return api.ExecuteRadar(b.RadarTargets, b.Router, b.MyConfig.SearxngURL, b.MyConfig.JinaAPIKey, b.MyConfig.MaxConcurrency, intent, b.MyConfig.TargetLanguage, docID, kw)
}

func (b *BuguModule) RunRadar(intent string) api.RadarResponse {
	return b.RunRadarForDoc("", intent)
}

func init() {
	module.Register(&BuguModule{})
}
