package module

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"Finnegan/core/models"
)

type mockLegacyModule struct {
	name string
	cmds []CommandDef
}

func (m *mockLegacyModule) Name() string                      { return m.name }
func (m *mockLegacyModule) ConfigTarget() interface{}         { return nil }
func (m *mockLegacyModule) Init(ctx *Context) error           { return nil }
func (m *mockLegacyModule) Bindings() []interface{}           { return nil }
func (m *mockLegacyModule) GetCommands() []CommandDef         { return m.cmds }
func (m *mockLegacyModule) ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error) {
	return nil, nil
}

type mockHostAPI struct {
	mu           sync.Mutex
	spawnedCards []models.BabelCard
	completedIDs []string
	emitted      map[string]interface{}
}

func (h *mockHostAPI) SpawnCard(card models.BabelCard) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.spawnedCards = append(h.spawnedCards, card)
}

func (h *mockHostAPI) CompleteSession(cardID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.completedIDs = append(h.completedIDs, cardID)
}

func (h *mockHostAPI) EmitEvent(eventName string, data interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.emitted == nil {
		h.emitted = make(map[string]interface{})
	}
	h.emitted[eventName] = data
}

func TestBuildModuleManifest_BuiltInFallbacks(t *testing.T) {
	buguMod := &mockLegacyModule{
		name: "bugu",
		cmds: []CommandDef{{ID: "bugu:radar", Name: "布谷深网雷达", Trigger: "bugu"}},
	}
	buguManifest := BuildModuleManifest(buguMod)
	if buguManifest.Name != "bugu" {
		t.Fatalf("expected name bugu, got %s", buguManifest.Name)
	}
	if buguManifest.ThemeColor != "#f59e0b" {
		t.Errorf("expected ThemeColor #f59e0b, got %s", buguManifest.ThemeColor)
	}
	if buguManifest.BeaconType != "radar" {
		t.Errorf("expected BeaconType radar, got %s", buguManifest.BeaconType)
	}
	if buguManifest.PromotionStrategy == nil || !buguManifest.PromotionStrategy.BindMotherCard {
		t.Errorf("expected PromotionStrategy with BindMotherCard=true, got %+v", buguManifest.PromotionStrategy)
	}
	if len(buguManifest.Commands) != 1 || buguManifest.Commands[0].ID != "bugu:radar" {
		t.Errorf("expected Commands populated from CommandProvider, got %+v", buguManifest.Commands)
	}

	atprotoMod := &mockLegacyModule{name: "atproto"}
	atprotoManifest := BuildModuleManifest(atprotoMod)
	if atprotoManifest.ThemeColor != "#38bdf8" || atprotoManifest.BeaconType != "butterfly" {
		t.Errorf("unexpected atproto manifest defaults: %+v", atprotoManifest)
	}
}

func TestLoadExternalPlugins_DirectoryAndSingleBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a directory-based plugin: <tmpDir>/taohuo/
	taohuoDir := filepath.Join(tmpDir, "taohuo")
	if err := os.MkdirAll(taohuoDir, 0755); err != nil {
		t.Fatal(err)
	}
	pluginYML := `name: taohuo
version: "1.0.0"
theme_color: "#ec4899"
beacon_type: "radar"
promotion_strategy:
  icon: "default_book"
  tooltip: "收录淘货卡片"
  has_comment_prompt: false
  placeholder: ""
  bind_mother_card: false
card_types:
  - type: "taohuo"
commands:
  - id: "taohuo:hunt"
    name: "淘货寻宝"
    trigger: "taohuo"
    description: "全网寻宝"
`
	if err := os.WriteFile(filepath.Join(taohuoDir, "plugin.yml"), []byte(pluginYML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taohuoDir, "ui.js"), []byte("export default class TaohuoCard {}"), 0644); err != nil {
		t.Fatal(err)
	}

	scriptContent := `#!/bin/sh
read input_json
echo '{"event":"spawn_card","card":{"id":"th_001","title":"Vintage Lens","content":"50mm f1.4","source_url":"https://example.com/item1","echo_of":"mother_123"}}'
echo '{"event":"complete_session","cardId":"mother_123"}'
echo '{"status":"triggered","command":"taohuo:hunt"}'
`
	execPath := filepath.Join(taohuoDir, "taohuo")
	if err := os.WriteFile(execPath, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	// 2. Create a standalone single-file .plugin binary: <tmpDir>/standalone.plugin
	singlePluginScript := `#!/bin/sh
if [ "$1" = "--finnegan-manifest" ]; then
  echo '{"name":"standalone","themeColor":"#10b981","beaconType":"butterfly","commands":[{"id":"standalone:ping","name":"Ping","trigger":"ping"}]}'
  exit 0
fi
if [ "$1" = "--finnegan-ui" ]; then
  echo 'export const standaloneUI = true;'
  exit 0
fi
read input_json
echo '{"status":"pong"}'
`
	singlePluginPath := filepath.Join(tmpDir, "standalone.plugin")
	if err := os.WriteFile(singlePluginPath, []byte(singlePluginScript), 0755); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadExternalPlugins(tmpDir)
	if err != nil {
		t.Fatalf("LoadExternalPlugins failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 loaded external plugins, got %d", len(loaded))
	}

	reg := GetRegistry()
	taohuoMod, ok := reg["taohuo"]
	if !ok {
		t.Fatal("taohuo module was not registered in global registry")
	}

	manifest := BuildModuleManifest(taohuoMod)
	if !manifest.IsExternal || manifest.ThemeColor != "#ec4899" {
		t.Errorf("unexpected taohuo manifest: %+v", manifest)
	}
	if len(manifest.CardTypes) != 1 || manifest.CardTypes[0].UIEntry != "/api/plugins/taohuo/ui.js" {
		t.Errorf("expected auto-wired UIEntry /api/plugins/taohuo/ui.js, got %+v", manifest.CardTypes)
	}

	host := &mockHostAPI{}
	if err := taohuoMod.Init(&Context{
		LLMs: map[string]LLMConfig{"default": {Provider: "gemini", DefaultModel: "gemini-3.8-flash"}},
		Host: host,
	}); err != nil {
		t.Fatalf("taohuoMod.Init failed: %v", err)
	}

	cp, ok := taohuoMod.(CommandProvider)
	if !ok {
		t.Fatal("taohuoMod does not implement CommandProvider")
	}
	res, err := cp.ExecuteCommand(context.Background(), "taohuo:hunt", map[string]interface{}{"text": "camera"})
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}
	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["status"] != "triggered" {
		t.Errorf("unexpected command result: %+v", res)
	}

	if len(host.spawnedCards) != 1 {
		t.Fatalf("expected 1 spawned card via HostAPI, got %d", len(host.spawnedCards))
	}
	if host.spawnedCards[0].ID != "th_001" || host.spawnedCards[0].Origin != "taohuo" || host.spawnedCards[0].EchoOf != "mother_123" {
		t.Errorf("unexpected spawned card: %+v", host.spawnedCards[0])
	}
	if len(host.completedIDs) != 1 || host.completedIDs[0] != "mother_123" {
		t.Errorf("expected completed session mother_123, got %+v", host.completedIDs)
	}

	// Verify standalone .plugin UI bundle extraction
	standaloneMod, ok := reg["standalone"].(*ExternalModule)
	if !ok {
		t.Fatal("standalone module not found or not *ExternalModule")
	}
	uiBytes, err := standaloneMod.GetUIBundle()
	if err != nil {
		t.Fatalf("standaloneMod.GetUIBundle failed: %v", err)
	}
	if string(uiBytes) != "export const standaloneUI = true;\n" {
		t.Errorf("unexpected UI bundle from standalone.plugin: %q", string(uiBytes))
	}
}
