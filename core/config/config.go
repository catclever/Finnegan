package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type PointerConfig struct {
	Home string `yaml:"home"`
}

type Config struct {
	ExternalProjects   []string `yaml:"external_projects"`
	DebounceMs         int      `yaml:"debounce_ms"`
	BuguMaxConcurrency int      `yaml:"bugu_max_concurrency"`
}

func GetPointerConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get user home dir: %v", err)
		return "finnegan.yml"
	}
	return filepath.Join(home, "Library", "Application Support", "Finnegan", "finnegan.yml")
}

func LoadConfig() (*Config, string, error) {
	pointerPath := GetPointerConfigPath()
	pointerDir := filepath.Dir(pointerPath)
	
	if err := os.MkdirAll(pointerDir, 0755); err != nil {
		return nil, "", err
	}

	var pointer PointerConfig
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		pointer.Home = "~/Finnegan"
		defaultData, _ := yaml.Marshal(pointer)
		_ = os.WriteFile(pointerPath, defaultData, 0644)
	} else {
		if err := yaml.Unmarshal(data, &pointer); err != nil {
			pointer.Home = "~/Finnegan"
		}
		if pointer.Home == "" {
			pointer.Home = "~/Finnegan"
		}
	}

	actualHome := pointer.Home
	if strings.HasPrefix(actualHome, "~") {
		if uHome, err := os.UserHomeDir(); err == nil {
			actualHome = filepath.Join(uHome, actualHome[1:])
		}
	}
	actualHome, _ = filepath.Abs(actualHome)

	// Create skeleton
	os.MkdirAll(filepath.Join(actualHome, "Scattering"), 0755)
	os.MkdirAll(filepath.Join(actualHome, "Weaving"), 0755)
	os.MkdirAll(filepath.Join(actualHome, ".finnegan", "bases"), 0755)

	configPath := filepath.Join(actualHome, ".finnegan", "config.yml")
	var cfg Config
	cData, err := os.ReadFile(configPath)
	if err != nil {
		cfg = Config{
			ExternalProjects:   []string{},
			DebounceMs:         300,
			BuguMaxConcurrency: 1,
		}
		defaultCfg, _ := yaml.Marshal(cfg)
		_ = os.WriteFile(configPath, defaultCfg, 0644)
	} else {
		if err := yaml.Unmarshal(cData, &cfg); err != nil {
			return nil, "", err
		}
	}

	if cfg.DebounceMs == 0 {
		cfg.DebounceMs = 300
	}
	if cfg.BuguMaxConcurrency < 1 {
		cfg.BuguMaxConcurrency = 1
	}

	var resolvedExternal []string
	for _, p := range cfg.ExternalProjects {
		if strings.HasPrefix(p, "~") {
			if uHome, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(uHome, p[1:])
			}
		}
		abs, err := filepath.Abs(p)
		if err == nil {
			resolvedExternal = append(resolvedExternal, abs)
		}
	}
	cfg.ExternalProjects = resolvedExternal

	return &cfg, actualHome, nil
}

func DeriveWatchDirs(home string, externalProjects []string) []string {
	dirs := []string{
		filepath.Join(home, "Scattering"),
		filepath.Join(home, "Weaving"),
	}
	for _, ep := range externalProjects {
		if stat, err := os.Stat(ep); err == nil && stat.IsDir() {
			marker := filepath.Join(ep, ".finnegan-project")
			if _, err := os.Stat(marker); err == nil {
				dirs = append(dirs, ep)
			}
		}
	}
	return dirs
}
