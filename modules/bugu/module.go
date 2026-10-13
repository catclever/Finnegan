package bugu

import (
	"Finnegan/modules/bugu/api"
	"Finnegan/modules/bugu/config"
	"Finnegan/modules/bugu/llm"
	"Finnegan/modules/bugu/models"
	"Finnegan/core/module"
	"Finnegan/core/bus"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type BuguConfig struct {
	SearxngURL     string `yaml:"searxng_url"`
	MaxConcurrency int    `yaml:"max_concurrency"`
	FallbackKey    string `yaml:"fallback_key"`
	JinaAPIKey     string `yaml:"jina_api_key"`
	TargetLanguage string `yaml:"target_language"`
	LLMID          string `yaml:"llm_id"`
}

type BuguModule struct {
	MyConfig     *BuguConfig
	Router       *llm.Router
	RadarTargets *config.Config
}

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
	cfg, err := config.LoadConfig("bugu/radar_targets.yml")
	if err != nil {
		cfg, err = config.LoadConfig("radar_targets.yml")
	}
	if err != nil {
		// Just a fallback for now
		cfg = &config.Config{Targets: []models.Target{}, TargetMap: make(map[string]models.Target)}
	}
	b.RadarTargets = cfg

	// Get specific LLM from core context
	llmID := b.MyConfig.LLMID
	if llmID == "" {
		llmID = "default"
	}
	
	globalLLM := ctx.LLMs[llmID]
	
	finalLLM := module.LLMConfig{
		Provider:     globalLLM.Provider,
		APIKey:       globalLLM.APIKey,
		BaseURL:      globalLLM.BaseURL,
		DefaultModel: globalLLM.DefaultModel,
	}

	// Override API Key tier
	if finalLLM.APIKey == "" {
		if b.MyConfig.FallbackKey != "" {
			finalLLM.APIKey = b.MyConfig.FallbackKey
		}
	}

	// Router initialization using final LLM config
	b.Router = llm.NewRouter(cfg, finalLLM)

	// Subscribe to new thoughts to automatically run Radar
	thoughtCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "bugu-module")
	go func() {
		for event := range thoughtCh {
			if data, ok := event.Data.(bus.EventThoughtCapturedData); ok {
				log.Printf("[Bugu Module] Triggered automatically by new thought: %s", data.Text)
				// Run radar asynchronously
				go b.RunRadar(data.Text)
			}
		}
	}()

	return nil
}

func (b *BuguModule) Bindings() []interface{} {
	return []interface{}{
		b,
	}
}

// Wails exposed methods

func (b *BuguModule) GetRadarTargets() []models.Target {
	if b.RadarTargets == nil {
		return []models.Target{}
	}
	return b.RadarTargets.Targets
}

func (b *BuguModule) RunRadar(intent string) api.RadarResponse {
	if b.RadarTargets == nil || b.Router == nil {
		return api.RadarResponse{
			Status: "error",
			Error:  "Bugu configuration is not loaded",
		}
	}
	return api.ExecuteRadar(b.RadarTargets, b.Router, b.MyConfig.SearxngURL, b.MyConfig.JinaAPIKey, b.MyConfig.MaxConcurrency, intent, b.MyConfig.TargetLanguage)
}

func init() {
	module.Register(&BuguModule{})
}
