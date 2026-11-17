package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigSkeletonAndPointer(t *testing.T) {
	// Setup temp home
	tempHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	cfg, home, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expectedHome := filepath.Join(tempHome, "Documents", "Finnegan")
	if home != expectedHome {
		t.Errorf("Expected home %s, got %s", expectedHome, home)
	}

	// Check skeleton
	dirs := []string{
		filepath.Join(expectedHome, "Scattering"),
		filepath.Join(expectedHome, "Weaving"),
		filepath.Join(expectedHome, ".finnegan", "bases"),
	}
	for _, d := range dirs {
		if stat, err := os.Stat(d); err != nil || !stat.IsDir() {
			t.Errorf("Expected directory %s to exist", d)
		}
	}

	if cfg.DebounceMs != 300 {
		t.Errorf("Expected debounce 300, got %d", cfg.DebounceMs)
	}
}

func TestLLMConfigParsing(t *testing.T) {
	yamlData := []byte(`
llms:
  default:
    provider: gemini
    api_key: "AIzaSy_TEST_KEY"
    default_model: "gemini-3.8-flash"
`)
	tempHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", origHome)

	pointerPath := GetPointerConfigPath()
	os.MkdirAll(filepath.Dir(pointerPath), 0755)
	os.WriteFile(pointerPath, yamlData, 0644)

	cfg, _, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	defLLM, ok := cfg.LLMs["default"]
	if !ok {
		t.Fatalf("Expected 'default' LLM in cfg.LLMs")
	}
	if defLLM.APIKey != "AIzaSy_TEST_KEY" {
		t.Errorf("Expected APIKey 'AIzaSy_TEST_KEY', got '%s'", defLLM.APIKey)
	}
	if defLLM.DefaultModel != "gemini-3.8-flash" {
		t.Errorf("Expected DefaultModel 'gemini-3.8-flash', got '%s'", defLLM.DefaultModel)
	}
	if defLLM.Provider != "gemini" {
		t.Errorf("Expected Provider 'gemini', got '%s'", defLLM.Provider)
	}
}

func TestDeriveWatchDirs(t *testing.T) {
	home := t.TempDir()
	ext1 := filepath.Join(t.TempDir(), "ext1")
	ext2 := filepath.Join(t.TempDir(), "ext2")
	os.MkdirAll(ext1, 0755)
	os.MkdirAll(ext2, 0755)
	
	// Create marker in ext1, not ext2
	os.WriteFile(filepath.Join(ext1, ".finnegan-project"), []byte(""), 0644)

	dirs := DeriveWatchDirs(home, []string{ext1, ext2})
	
	expected := []string{
		filepath.Join(home, "Scattering"),
		filepath.Join(home, "Weaving"),
		ext1,
	}
	
	if len(dirs) != len(expected) {
		t.Errorf("Expected %d watch dirs, got %d", len(expected), len(dirs))
	}
	for i, d := range expected {
		if i < len(dirs) && dirs[i] != d {
			t.Errorf("Expected %s at %d, got %s", d, i, dirs[i])
		}
	}
}

func TestJinaConfigParsing(t *testing.T) {
	// Case 1: Custom APIKey and Custom BaseURL via LoadConfig
	t.Run("CustomKeyAndBaseURL", func(t *testing.T) {
		yamlData := []byte(`
jina:
  api_key: "jina_test_key_custom"
  base_url: "https://custom.jina.ai/"
`)
		tempHome := t.TempDir()
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", origHome)

		pointerPath := GetPointerConfigPath()
		os.MkdirAll(filepath.Dir(pointerPath), 0755)
		os.WriteFile(pointerPath, yamlData, 0644)

		cfg, _, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Jina.APIKey != "jina_test_key_custom" {
			t.Errorf("Expected APIKey 'jina_test_key_custom', got '%s'", cfg.Jina.APIKey)
		}
		if cfg.Jina.BaseURL != "https://custom.jina.ai/" {
			t.Errorf("Expected BaseURL 'https://custom.jina.ai/', got '%s'", cfg.Jina.BaseURL)
		}
	})

	// Case 2: Only APIKey specified; BaseURL should default to https://r.jina.ai/
	t.Run("DefaultBaseURLWhenOmittedInYAML", func(t *testing.T) {
		yamlData := []byte(`
jina:
  api_key: "jina_only_key"
`)
		tempHome := t.TempDir()
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", origHome)

		pointerPath := GetPointerConfigPath()
		os.MkdirAll(filepath.Dir(pointerPath), 0755)
		os.WriteFile(pointerPath, yamlData, 0644)

		cfg, _, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Jina.APIKey != "jina_only_key" {
			t.Errorf("Expected APIKey 'jina_only_key', got '%s'", cfg.Jina.APIKey)
		}
		if cfg.Jina.BaseURL != "https://r.jina.ai/" {
			t.Errorf("Expected default BaseURL 'https://r.jina.ai/', got '%s'", cfg.Jina.BaseURL)
		}
	})

	// Case 3: No jina block in YAML; BaseURL should still default to https://r.jina.ai/
	t.Run("DefaultBaseURLWhenNoJinaBlock", func(t *testing.T) {
		yamlData := []byte(`
debounce_ms: 500
`)
		tempHome := t.TempDir()
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", origHome)

		pointerPath := GetPointerConfigPath()
		os.MkdirAll(filepath.Dir(pointerPath), 0755)
		os.WriteFile(pointerPath, yamlData, 0644)

		cfg, _, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Jina.BaseURL != "https://r.jina.ai/" {
			t.Errorf("Expected default BaseURL 'https://r.jina.ai/', got '%s'", cfg.Jina.BaseURL)
		}
	})

	// Case 4: Environment variable fallback for JINA_API_KEY
	t.Run("EnvJinaAPIKeyFallback", func(t *testing.T) {
		origEnv := os.Getenv("JINA_API_KEY")
		os.Setenv("JINA_API_KEY", "env_jina_key_xyz")
		defer os.Setenv("JINA_API_KEY", origEnv)

		yamlData := []byte(`
debounce_ms: 300
`)
		tempHome := t.TempDir()
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", origHome)

		pointerPath := GetPointerConfigPath()
		os.MkdirAll(filepath.Dir(pointerPath), 0755)
		os.WriteFile(pointerPath, yamlData, 0644)

		cfg, _, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Jina.APIKey != "env_jina_key_xyz" {
			t.Errorf("Expected APIKey 'env_jina_key_xyz', got '%s'", cfg.Jina.APIKey)
		}
		if cfg.Jina.BaseURL != "https://r.jina.ai/" {
			t.Errorf("Expected default BaseURL 'https://r.jina.ai/', got '%s'", cfg.Jina.BaseURL)
		}
	})

	// Case 5: Direct YAML unmarshal into Config
	t.Run("DirectYAMLUnmarshal", func(t *testing.T) {
		var cfg Config
		err := yaml.Unmarshal([]byte(`
jina:
  api_key: "direct_key"
`), &cfg)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if cfg.Jina.APIKey != "direct_key" {
			t.Errorf("Expected APIKey 'direct_key', got '%s'", cfg.Jina.APIKey)
		}
		if cfg.Jina.BaseURL != "https://r.jina.ai/" {
			t.Errorf("Expected default BaseURL 'https://r.jina.ai/', got '%s'", cfg.Jina.BaseURL)
		}
	})
}

func TestDefaultLLMExplicitConfig(t *testing.T) {
	t.Run("default_llm mapping to default", func(t *testing.T) {
		yamlData := []byte(`
default_llm: "satan_gemini"

llms:
  satan_gemini:
    provider: gemini
    api_key: "satan_secret_key"
    default_model: "gemini-3.8-flash"
  work_buddy:
    provider: openai
    api_key: "work_key"
`)
		tempHome := t.TempDir()
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", origHome)

		pointerPath := GetPointerConfigPath()
		os.MkdirAll(filepath.Dir(pointerPath), 0755)
		os.WriteFile(pointerPath, yamlData, 0644)

		cfg, _, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.GetDefaultLLMID() != "satan_gemini" {
			t.Errorf("Expected GetDefaultLLMID 'satan_gemini', got '%s'", cfg.GetDefaultLLMID())
		}

		defLLM, ok := cfg.GetDefaultLLM()
		if !ok || defLLM.APIKey != "satan_secret_key" {
			t.Errorf("Expected GetDefaultLLM APIKey 'satan_secret_key', got '%s'", defLLM.APIKey)
		}

		// Also check that cfg.LLMs["default"] was automatically populated for backwards compatibility
		aliasedDef, ok := cfg.LLMs["default"]
		if !ok || aliasedDef.APIKey != "satan_secret_key" {
			t.Errorf("Expected cfg.LLMs['default'] to be aliased, got %+v", aliasedDef)
		}
	})

	t.Run("default_llm_id alternative syntax", func(t *testing.T) {
		yamlData := []byte(`
default_llm_id: "work_buddy"

llms:
  work_buddy:
    provider: openai
    api_key: "work_key"
`)
		tempHome := t.TempDir()
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", origHome)

		pointerPath := GetPointerConfigPath()
		os.MkdirAll(filepath.Dir(pointerPath), 0755)
		os.WriteFile(pointerPath, yamlData, 0644)

		cfg, _, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.GetDefaultLLMID() != "work_buddy" {
			t.Errorf("Expected GetDefaultLLMID 'work_buddy', got '%s'", cfg.GetDefaultLLMID())
		}
	})
}
