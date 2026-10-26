package config

import (
	"testing"
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
}
