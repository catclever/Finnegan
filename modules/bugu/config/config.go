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
