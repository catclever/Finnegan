package config

import (
	"fmt"
	"os"

	"path/filepath"
	"strings"

	"Finnegan/modules/bugu/models"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Targets      []models.Target
	TargetMap    map[string]models.Target
	ConfigPath   string
	I18nFilePath string
}

// BuguConfig defines the configuration options for the Bugu radar module.
type BuguConfig struct {
	SearxngURL     string `yaml:"searxng_url" json:"searxng_url"`
	MaxConcurrency int    `yaml:"max_concurrency" json:"max_concurrency"`
	FallbackKey    string `yaml:"fallback_key" json:"fallback_key"`
	APIKey         string `yaml:"api_key" json:"api_key"`
	JinaAPIKey     string `yaml:"jina_api_key" json:"jina_api_key"`
	TargetLanguage string `yaml:"target_language" json:"target_language"`
	LLMID          string `yaml:"llm_id" json:"llm_id"`
	DefaultModel   string `yaml:"default_model" json:"default_model"`
	RouterModel    string `yaml:"router_model" json:"router_model"`
	DistillerModel string `yaml:"distiller_model" json:"distiller_model"`
}

// GetRouterModel returns the configured router model, smoothly falling back to DefaultModel or "gemini-3.8-flash".
func (c *BuguConfig) GetRouterModel() string {
	if c != nil && strings.TrimSpace(c.RouterModel) != "" {
		return strings.TrimSpace(c.RouterModel)
	}
	if c != nil && strings.TrimSpace(c.DefaultModel) != "" {
		return strings.TrimSpace(c.DefaultModel)
	}
	return "gemini-3.8-flash"
}

// GetDistillerModel returns the configured distiller model, smoothly falling back to DefaultModel or "gemini-3.8-flash".
func (c *BuguConfig) GetDistillerModel() string {
	if c != nil && strings.TrimSpace(c.DistillerModel) != "" {
		return strings.TrimSpace(c.DistillerModel)
	}
	if c != nil && strings.TrimSpace(c.DefaultModel) != "" {
		return strings.TrimSpace(c.DefaultModel)
	}
	return "gemini-3.8-flash"
}

type TagTranslator interface {
	TranslateTags(tags []string) (map[string]map[string]string, error)
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var root models.RadarTargets
	err = yaml.Unmarshal(data, &root)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	targetMap := make(map[string]models.Target)
	for _, t := range root.Targets {
		targetMap[t.ID] = t
	}

	i18nPath := filepath.Join(filepath.Dir(path), "tags_i18n.json")
	_ = LoadCustomTranslations(i18nPath)

	return &Config{
		Targets:      root.Targets,
		TargetMap:    targetMap,
		ConfigPath:   path,
		I18nFilePath: i18nPath,
	}, nil
}

// FindUntranslatedTags finds all unique tags that are not yet translated in default or custom dictionaries.
func (c *Config) FindUntranslatedTags() []string {
	seen := make(map[string]bool)
	var missing []string

	for _, t := range c.Targets {
		for _, tag := range t.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" || seen[tag] {
				continue
			}
			seen[tag] = true
			if !HasTagTranslation(tag) {
				missing = append(missing, tag)
			}
		}
	}
	return missing
}

// AutoEnrich translates missing tags via TagTranslator and persists them to I18nFilePath.
func (c *Config) AutoEnrich(translator TagTranslator) error {
	missing := c.FindUntranslatedTags()
	if len(missing) == 0 {
		return nil
	}

	newTranslations, err := translator.TranslateTags(missing)
	if err != nil {
		return err
	}

	return MergeCustomTranslations(newTranslations, c.I18nFilePath)
}

// GetDistillerModel returns empty for now to fallback to LLM DefaultModel
func (c *Config) GetDistillerModel() string {
	return ""
}

func (c *Config) GetTargetCatalog() string {
	// Generates a compact catalog of tags and IDs for the LLM prompt.
	// Automatically translates tags to each target's native language so the LLM
	// matches foreign targets natively without language bias.
	catalog := ""
	for _, t := range c.Targets {
		primaryLang := "en"
		if len(t.Languages) > 0 {
			primaryLang = t.Languages[0]
		}
		translatedTags := TranslateTags(t.Tags, primaryLang)
		catalog += fmt.Sprintf("ID: %s | Type: %s | Langs: %v | Tags: %v\n", t.ID, t.Type, t.Languages, translatedTags)
	}
	return catalog
}
