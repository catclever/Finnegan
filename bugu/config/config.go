package config

import (
	"fmt"
	"os"

	"Finnegan/bugu/models"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Targets   []models.Target
	TargetMap map[string]models.Target
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

	return &Config{
		Targets:   root.Targets,
		TargetMap: targetMap,
	}, nil
}

func (c *Config) GetTargetCatalog() string {
	// Generates a compact catalog of tags and IDs for the LLM prompt.
	// We map this out so the LLM knows what domains exist without reading 500 URLs.
	catalog := ""
	for _, t := range c.Targets {
		catalog += fmt.Sprintf("ID: %s | Type: %s | Langs: %v | Tags: %v\n", t.ID, t.Type, t.Languages, t.Tags)
	}
	return catalog
}
