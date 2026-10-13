package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"Finnegan/core/module"

	"gopkg.in/yaml.v3"
)

type Config struct {
	OverrideConfig   string                          `yaml:"override_config"`
	DataDir          string                          `yaml:"data_dir"`
	ExternalProjects []string                        `yaml:"external_projects"`
	DebounceMs       int                             `yaml:"debounce_ms"`
	LLMs             map[string]module.LLMConfig     `yaml:"llms"`
	SingleLLM        *module.LLMConfig               `yaml:"llm"`
	RawModules       map[string]yaml.Node            `yaml:",inline"`
}

func GetPointerConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get user home dir: %v", err)
		return "finnegan.yml"
	}
	return filepath.Join(home, "Library", "Application Support", "Finnegan", "finnegan.yml")
}

func resolvePath(p string) string {
	if strings.HasPrefix(p, "~") {
		if uHome, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(uHome, p[1:])
		}
	}
	abs, err := filepath.Abs(p)
	if err == nil {
		return abs
	}
	return p
}

func LoadConfig() (*Config, string, error) {
	appSupportPath := GetPointerConfigPath()
	pointerDir := filepath.Dir(appSupportPath)

	if err := os.MkdirAll(pointerDir, 0755); err != nil {
		return nil, "", err
	}

	var data []byte
	var err error

	data, err = os.ReadFile(appSupportPath)
	if err != nil {
		cfg := Config{
			DataDir:    "~/Documents/Finnegan",
			DebounceMs: 300,
			LLMs: map[string]module.LLMConfig{
				"default": {
					Provider:     "gemini",
					APIKey:       "",
					DefaultModel: "gemini-3.8-flash",
				},
			},
		}
		data, _ = yaml.Marshal(cfg)
		_ = os.WriteFile(appSupportPath, data, 0644)
	}

	// Check for override_config pointer
	var temp struct {
		OverrideConfig string `yaml:"override_config"`
	}
	_ = yaml.Unmarshal(data, &temp)

	if temp.OverrideConfig != "" {
		resolvedOverride := resolvePath(temp.OverrideConfig)
		overrideData, err := os.ReadFile(resolvedOverride)
		if err == nil {
			data = overrideData // Success! Override data
			log.Printf("Loaded config from override: %s", resolvedOverride)
		} else {
			log.Printf("Warning: override_config %s not found. Falling back to local.", resolvedOverride)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", err
	}

	if cfg.LLMs == nil {
		cfg.LLMs = make(map[string]module.LLMConfig)
	}
	if cfg.SingleLLM != nil {
		def, exists := cfg.LLMs["default"]
		if !exists || def.APIKey == "" {
			cfg.LLMs["default"] = *cfg.SingleLLM
		}
	}
	// Fallback to env GEMINI_API_KEY if default api_key is empty
	if def, exists := cfg.LLMs["default"]; exists && def.APIKey == "" {
		if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
			def.APIKey = envKey
			cfg.LLMs["default"] = def
		}
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "~/Documents/Finnegan"
	}
	actualDataDir := resolvePath(cfg.DataDir)

	// Create skeleton
	os.MkdirAll(filepath.Join(actualDataDir, "Scattering"), 0755)
	os.MkdirAll(filepath.Join(actualDataDir, "Weaving"), 0755)
	os.MkdirAll(filepath.Join(actualDataDir, ".finnegan", "bases"), 0755)

	if cfg.DebounceMs == 0 {
		cfg.DebounceMs = 300
	}

	// Dynamic register bindings for modules
	registry := module.GetRegistry()
	for name, m := range registry {
		if node, exists := cfg.RawModules[name]; exists {
			cfgTarget := m.ConfigTarget()
			if cfgTarget != nil {
				_ = node.Decode(cfgTarget)
			}
		}
	}

	var resolvedExternal []string
	for _, p := range cfg.ExternalProjects {
		resolvedExternal = append(resolvedExternal, resolvePath(p))
	}
	cfg.ExternalProjects = resolvedExternal

	return &cfg, actualDataDir, nil
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
