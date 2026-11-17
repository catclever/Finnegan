package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApp_GetLLMConfig_SmartFallback(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "Library", "Application Support", "Finnegan")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "finnegan.yml")
	content := `
override_config: ""
data_dir: "~/Documents/Finnegan"

llms:
  deepseek_workbuddy:
    provider: openai
    base_url: "http://172.16.51.7:48787/v1"
    default_model: "deepseek-v4.1-flash"
    api_key: "sk-mock-key"
  satan_gemini:
    provider: gemini
    api_key: "mock-gemini-key-123"
    default_model: "gemini-3.8-flash"
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	app := &App{}

	// Case 1: No default, but satan_gemini exists with provider gemini -> should pick satan_gemini
	cfg := app.getLLMConfig()
	if cfg.Provider != "gemini" {
		t.Errorf("expected provider gemini, got %s", cfg.Provider)
	}
	if cfg.APIKey != "mock-gemini-key-123" {
		t.Errorf("expected APIKey mock-gemini-key-123, got %s", cfg.APIKey)
	}
	if cfg.DefaultModel != "gemini-3.8-flash" {
		t.Errorf("expected model gemini-3.8-flash, got %s", cfg.DefaultModel)
	}

	// Case 2: Preferred ID specified
	deepseekCfg := app.getLLMConfig("deepseek_workbuddy")
	if deepseekCfg.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", deepseekCfg.Provider)
	}
}

func TestApp_GetLLMConfig_KeywordsBlockPreference(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "Library", "Application Support", "Finnegan")
	_ = os.MkdirAll(configDir, 0755)

	configFile := filepath.Join(configDir, "finnegan.yml")
	content := `
llms:
  gemini_one:
    provider: gemini
    api_key: "key-1"
  gemini_two:
    provider: gemini
    api_key: "key-2"

keywords:
  llm_id: "gemini_two"
`
	_ = os.WriteFile(configFile, []byte(content), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	app := &App{}
	cfg := app.getLLMConfig()
	if cfg.APIKey != "key-2" {
		t.Errorf("expected keywords.llm_id preference key-2, got %s", cfg.APIKey)
	}
}

func TestApp_GetLLMConfig_GlobalDefaultLLM(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "Library", "Application Support", "Finnegan")
	_ = os.MkdirAll(configDir, 0755)

	configFile := filepath.Join(configDir, "finnegan.yml")
	content := `
default_llm: "satan_gemini"

llms:
  free_gemini:
    provider: gemini
    api_key: "free-key"
  satan_gemini:
    provider: gemini
    api_key: "satan-key"
`
	_ = os.WriteFile(configFile, []byte(content), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	app := &App{}
	cfg := app.getLLMConfig()
	if cfg.APIKey != "satan-key" {
		t.Errorf("expected global default_llm to select satan_gemini (satan-key), got %s", cfg.APIKey)
	}
}
