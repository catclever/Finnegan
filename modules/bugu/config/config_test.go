package config

import (
	"os"
	"testing"

	coreConfig "Finnegan/core/config"
)

func TestBuguConfig_ModelFallback(t *testing.T) {
	// 1. Both empty: smoothly fallback to global default "gemini-3.8-flash"
	cfgEmpty := &BuguConfig{}
	if m := cfgEmpty.GetRouterModel(); m != "gemini-3.8-flash" {
		t.Errorf("expected default 'gemini-3.8-flash' for router, got %q", m)
	}
	if m := cfgEmpty.GetDistillerModel(); m != "gemini-3.8-flash" {
		t.Errorf("expected default 'gemini-3.8-flash' for distiller, got %q", m)
	}

	// 2. Nil receiver safety: smooth fallback
	var cfgNil *BuguConfig
	if m := cfgNil.GetRouterModel(); m != "gemini-3.8-flash" {
		t.Errorf("expected default 'gemini-3.8-flash' for nil router, got %q", m)
	}
	if m := cfgNil.GetDistillerModel(); m != "gemini-3.8-flash" {
		t.Errorf("expected default 'gemini-3.8-flash' for nil distiller, got %q", m)
	}

	// 3. DefaultModel set, router and distiller empty: both fallback to DefaultModel ("如果不写就是二者都是")
	cfgShared := &BuguConfig{
		DefaultModel: "gemini-2.5-pro",
	}
	if m := cfgShared.GetRouterModel(); m != "gemini-2.5-pro" {
		t.Errorf("expected DefaultModel 'gemini-2.5-pro' for router, got %q", m)
	}
	if m := cfgShared.GetDistillerModel(); m != "gemini-2.5-pro" {
		t.Errorf("expected DefaultModel 'gemini-2.5-pro' for distiller, got %q", m)
	}

	// 4. RouterModel set specifically, distiller empty: router uses specific, distiller falls back to DefaultModel
	cfgRouterOnly := &BuguConfig{
		DefaultModel: "gemini-2.5-pro",
		RouterModel:  "gemini-3.8-flash",
	}
	if m := cfgRouterOnly.GetRouterModel(); m != "gemini-3.8-flash" {
		t.Errorf("expected 'gemini-3.8-flash' for router, got %q", m)
	}
	if m := cfgRouterOnly.GetDistillerModel(); m != "gemini-2.5-pro" {
		t.Errorf("expected 'gemini-2.5-pro' for distiller, got %q", m)
	}

	// 5. DistillerModel set specifically, router empty: distiller uses specific, router falls back
	cfgDistillerOnly := &BuguConfig{
		DefaultModel:   "gemini-3.8-flash",
		DistillerModel: "gemini-1.5-pro",
	}
	if m := cfgDistillerOnly.GetRouterModel(); m != "gemini-3.8-flash" {
		t.Errorf("expected 'gemini-3.8-flash' for router, got %q", m)
	}
	if m := cfgDistillerOnly.GetDistillerModel(); m != "gemini-1.5-pro" {
		t.Errorf("expected 'gemini-1.5-pro' for distiller, got %q", m)
	}

	// 6. Both set specifically
	cfgSplit := &BuguConfig{
		DefaultModel:   "gemini-3.8-flash",
		RouterModel:    "gemini-flash-custom",
		DistillerModel: "gemini-pro-custom",
	}
	if m := cfgSplit.GetRouterModel(); m != "gemini-flash-custom" {
		t.Errorf("expected 'gemini-flash-custom' for router, got %q", m)
	}
	if m := cfgSplit.GetDistillerModel(); m != "gemini-pro-custom" {
		t.Errorf("expected 'gemini-pro-custom' for distiller, got %q", m)
	}

	// 7. Verify Config.GetRouterModel and Config.GetDistillerModel
	coreCfg := &Config{
		RouterModel:    "deepseek-v4.1-flash",
		DistillerModel: "gemini-3.8-flash",
	}
	if coreCfg.GetRouterModel() != "deepseek-v4.1-flash" {
		t.Errorf("expected 'deepseek-v4.1-flash', got %q", coreCfg.GetRouterModel())
	}
	if coreCfg.GetDistillerModel() != "gemini-3.8-flash" {
		t.Errorf("expected 'gemini-3.8-flash', got %q", coreCfg.GetDistillerModel())
	}

	// 8. Verify BuguConfig LLMID fields
	cfgWithLLMIDs := &BuguConfig{
		LLMID:          "default",
		RouterLLMID:    "deepseek_local",
		DistillerLLMID: "gemini_flash",
	}
	if cfgWithLLMIDs.RouterLLMID != "deepseek_local" || cfgWithLLMIDs.DistillerLLMID != "gemini_flash" {
		t.Errorf("unexpected LLMIDs: router=%q, distiller=%q", cfgWithLLMIDs.RouterLLMID, cfgWithLLMIDs.DistillerLLMID)
	}
}

func TestBuguConfig_JinaFallback(t *testing.T) {
	// 1. Explicit JinaAPIKey in BuguConfig takes precedence
	cfgExplicit := &BuguConfig{
		JinaAPIKey: "bugu-explicit-key",
	}
	globalJina := coreConfig.JinaConfig{
		APIKey:  "global-jina-key",
		BaseURL: "https://global.jina.ai/",
	}
	if key := cfgExplicit.GetJinaAPIKey(globalJina); key != "bugu-explicit-key" {
		t.Errorf("expected explicit key 'bugu-explicit-key', got %q", key)
	}

	// 2. Empty BuguConfig falls back to global JinaConfig
	cfgEmpty := &BuguConfig{}
	if key := cfgEmpty.GetJinaAPIKey(globalJina); key != "global-jina-key" {
		t.Errorf("expected fallback key 'global-jina-key', got %q", key)
	}
	if baseURL := cfgEmpty.GetJinaBaseURL(globalJina); baseURL != "https://global.jina.ai/" {
		t.Errorf("expected fallback baseURL 'https://global.jina.ai/', got %q", baseURL)
	}

	// 3. Fallback to JINA_API_KEY environment variable if global is not provided
	origEnv := os.Getenv("JINA_API_KEY")
	os.Setenv("JINA_API_KEY", "env-jina-12345")
	defer os.Setenv("JINA_API_KEY", origEnv)

	cfgEnv := &BuguConfig{}
	if key := cfgEnv.GetJinaAPIKey(); key != "env-jina-12345" {
		t.Errorf("expected env fallback key 'env-jina-12345', got %q", key)
	}
	if baseURL := cfgEnv.GetJinaBaseURL(); baseURL != "https://r.jina.ai/" {
		t.Errorf("expected default baseURL 'https://r.jina.ai/', got %q", baseURL)
	}
}
