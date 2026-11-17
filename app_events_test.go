package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/keywords"
	coremodels "Finnegan/core/models"
	"Finnegan/core/module"
	"Finnegan/core/watcher"
)

func TestFormatKeywordsYAML(t *testing.T) {
	profile := keywords.KeywordsProfile{
		"world":  []string{"Matrix", "Cyberspace"},
		"vision": []string{"SciFi"},
	}
	out := keywords.FormatKeywordsYAML(profile)
	if !strings.Contains(out, "keywords:") {
		t.Fatalf("expected keywords: in output, got: %s", out)
	}
	if !strings.Contains(out, "world:") || !strings.Contains(out, "Matrix") {
		t.Fatalf("expected world/Matrix in output, got: %s", out)
	}
}

func TestSetupWatcherCallbacks(t *testing.T) {
	app := &App{}
	storage := &WatcherStorage{}
	globalWatcher := watcher.GetWatcher()

	app.setupWatcherCallbacks(globalWatcher, storage)

	var changedID, changedData string
	var deletedID string
	var conflictID string
	var conflictPaths []string

	storage.SetOnChange(func(id, data string) {
		changedID = id
		changedData = data
	})
	storage.SetOnDelete(func(id string) {
		deletedID = id
	})
	storage.SetOnConflict(func(id string, paths []string) {
		conflictID = id
		conflictPaths = paths
	})

	_ = storage.SaveNote("note-1", "content-1")
	if changedID != "note-1" || changedData != "content-1" {
		t.Fatalf("SaveNote callback failed, got id=%s, data=%s", changedID, changedData)
	}

	_ = storage.DeleteNote("note-1")
	if deletedID != "note-1" {
		t.Fatalf("DeleteNote callback failed, got id=%s", deletedID)
	}

	storage.EmitConflict("note-conflict", []string{"path/a", "path/b"})
	if conflictID != "note-conflict" || len(conflictPaths) != 2 {
		t.Fatalf("EmitConflict callback failed, got id=%s, paths=%v", conflictID, conflictPaths)
	}
}

func TestOnThoughtCapturedEvent(t *testing.T) {
	app := &App{}
	// Should execute gracefully without panic when ctx is nil
	app.onThoughtCapturedEvent(bus.Message{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:      "thought-test",
			Text:       "Testing thought capture",
			CapturedAt: time.Now(),
		},
	})
}

type mockActionModule struct{}

func (m *mockActionModule) Name() string              { return "mock_action_mod" }
func (m *mockActionModule) ConfigTarget() interface{} { return nil }
func (m *mockActionModule) Init(ctx *module.Context) error {
	return nil
}
func (m *mockActionModule) Bindings() []interface{} { return nil }
func (m *mockActionModule) ExecuteAction(ctx context.Context, actionID string, payload map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"action": actionID,
		"echo":   payload["msg"],
	}, nil
}

func TestNormalizeAndSpawnBabelCard(t *testing.T) {
	app := &App{
		CardManager: NewCardManager(),
	}
	defer app.CardManager.Close()

	// 1. Test coremodels.BabelCard
	app.onBabelCardSpawned(bus.Message{
		Type: bus.EventBabelCardSpawned,
		Data: coremodels.BabelCard{
			ID:           "bc_1",
			Origin:       "bugu",
			Title:        "Deep Web Echo",
			OriginalText: "Hello from Bugu",
			EchoOf:       "mother_1",
		},
	})

	// 2. Test legacy Post-like struct via reflection/JSON normalization without importing module sub-packages
	type Post struct {
		Content      string
		Origin       string
		SourceURL    string
		AuthorName   string
		OriginalText string
		EchoOf       string
		LikeCount    int `json:"likeCount,omitempty"`
	}
	app.onBabelCardSpawned(bus.Message{
		Type: bus.EventBabelCardSpawned,
		Data: Post{
			Content:      "> [@alice]\n\nHello Bluesky",
			OriginalText: "Hello Bluesky",
			SourceURL:    "https://bsky.app/profile/alice/post/123",
			AuthorName:   "alice",
			EchoOf:       "mother_2",
			LikeCount:    7,
		},
	})

	tempCards := app.GetTempCards()
	if len(tempCards) != 2 {
		t.Fatalf("expected 2 injected temp cards, got %d", len(tempCards))
	}
}

func TestGetModuleManifestsAndExecuteModuleAction(t *testing.T) {
	module.Register(&mockActionModule{})
	app := &App{}

	manifests := app.GetModuleManifests()
	if len(manifests) == 0 {
		t.Fatal("expected non-empty module manifests")
	}

	res, err := app.ExecuteModuleAction("mock_action_mod", "ping", map[string]interface{}{"msg": "hello"})
	if err != nil {
		t.Fatalf("ExecuteModuleAction failed: %v", err)
	}
	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["echo"] != "hello" {
		t.Errorf("unexpected action result: %+v", res)
	}
}

func TestPluginUIAssetHandler(t *testing.T) {
	tmpHome := t.TempDir()
	pluginDir := filepath.Join(tmpHome, ".finnegan", "plugins", "demo_plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	uiContent := "export default function DemoCard() {}"
	if err := os.WriteFile(filepath.Join(pluginDir, "ui.js"), []byte(uiContent), 0644); err != nil {
		t.Fatal(err)
	}

	app := &App{Home: tmpHome}
	handler := createAssetHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/demo_plugin/ui.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
		t.Errorf("expected javascript Content-Type, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != uiContent {
		t.Errorf("unexpected body: %q", rec.Body.String())
	}
}

func TestOnBabelCardSpawned_PreservesTypeMetaTagsImages(t *testing.T) {
	app := &App{
		CardManager: NewCardManager(),
	}
	defer app.CardManager.Close()

	app.onBabelCardSpawned(bus.Message{
		Type: bus.EventBabelCardSpawned,
		Data: coremodels.BabelCard{
			ID:      "custom_bc_1",
			Type:    "taohuo",
			Origin:  "taohuo",
			Title:   "Custom Plugin Card",
			Content: "Distilled custom content",
			Tags:    []string{"vintage", "lens"},
			Images:  []string{"lens.jpg"},
			Meta: map[string]interface{}{
				"price":  "¥1280",
				"rating": 4.8,
			},
		},
	})

	app.onBabelCardSpawned(bus.Message{
		Type: bus.EventBabelCardSpawned,
		Data: map[string]interface{}{
			"id":      "custom_bc_2",
			"type":    "wishlist",
			"origin":  "shop",
			"title":   "Map Payload Card",
			"content": "Map content",
			"tags":    []string{"wish"},
			"images":  []string{"wish.png"},
			"meta": map[string]interface{}{
				"sku": "W-100",
			},
		},
	})

	tempCards := app.GetTempCards()
	if len(tempCards) != 2 {
		t.Fatalf("expected 2 temp cards, got %d", len(tempCards))
	}

	byID := make(map[string]Card)
	for _, c := range tempCards {
		byID[c.ID] = c
	}

	c1, ok := byID["custom_bc_1"]
	if !ok {
		t.Fatal("missing custom_bc_1")
	}
	if c1.Type != CardType("taohuo") {
		t.Errorf("expected Type 'taohuo', got %q", c1.Type)
	}
	if len(c1.Tags) != 2 || c1.Tags[0] != "vintage" || len(c1.Images) != 1 || c1.Images[0] != "lens.jpg" {
		t.Errorf("unexpected tags/images on c1: tags=%v images=%v", c1.Tags, c1.Images)
	}
	if c1.Meta["price"] != "¥1280" || c1.Meta["rating"] != 4.8 {
		t.Errorf("unexpected Meta on c1: %#v", c1.Meta)
	}

	c2, ok := byID["custom_bc_2"]
	if !ok {
		t.Fatal("missing custom_bc_2")
	}
	if c2.Type != CardType("wishlist") || c2.Meta["sku"] != "W-100" {
		t.Errorf("unexpected c2 type/meta: type=%q meta=%#v", c2.Type, c2.Meta)
	}
}

type mockCaptureModule struct {
	name     string
	cardType string
	handler  func(ctx context.Context, req module.CaptureRequest) (*module.CaptureResult, error)
}

func (m *mockCaptureModule) Name() string              { return m.name }
func (m *mockCaptureModule) ConfigTarget() interface{} { return nil }
func (m *mockCaptureModule) Init(ctx *module.Context) error {
	return nil
}
func (m *mockCaptureModule) Bindings() []interface{} { return nil }
func (m *mockCaptureModule) GetManifest() module.ModuleManifest {
	return module.ModuleManifest{
		Name: m.name,
		CardTypes: []module.CardTypeSpec{
			{Type: m.cardType, Name: "MockCard"},
		},
	}
}
func (m *mockCaptureModule) CaptureCard(ctx context.Context, req module.CaptureRequest) (*module.CaptureResult, error) {
	return m.handler(ctx, req)
}

func TestApp_CaptureModuleCard_HostAndPluginManaged(t *testing.T) {
	tmpHome := t.TempDir()
	scatteringDir := filepath.Join(tmpHome, "Scattering")
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a reference card so @RefCard resolves into threads
	refCardMD := "---\nid: ref_card_001\ntype: note\ntitle: RefCard\n---\n\nReference body\n"
	if err := os.WriteFile(filepath.Join(scatteringDir, "ref.md"), []byte(refCardMD), 0644); err != nil {
		t.Fatal(err)
	}

	var capturedReq module.CaptureRequest
	mod := &mockCaptureModule{
		name:     "mock_capture_host",
		cardType: "gadget",
		handler: func(ctx context.Context, req module.CaptureRequest) (*module.CaptureResult, error) {
			capturedReq = req
			return &module.CaptureResult{
				Title:   "Pocket Synth",
				Body:    req.CleanText,
				FileExt: ".gadget.md",
				Meta: map[string]interface{}{
					"price":  "¥899",
					"rating": 5,
					"sku":    "OP-1",
				},
			}, nil
		},
	}
	module.Register(mod)
	t.Cleanup(func() {
		module.ResetExternalPlugins()
	})

	app := &App{
		Home:        tmpHome,
		CardManager: NewCardManager(),
	}
	defer app.CardManager.Close()

	evCh := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_capture_module_card")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_capture_module_card")

	docID, err := app.CaptureModuleCard(":gadget", "@RefCard Great synth #music #gear", nil, map[string]interface{}{"sourceHint": "manual"})
	if err != nil {
		t.Fatalf("CaptureModuleCard failed: %v", err)
	}
	if docID == "" {
		t.Fatal("expected non-empty docID")
	}

	if capturedReq.CardType != "gadget" {
		t.Errorf("expected normalized cardType 'gadget', got %q", capturedReq.CardType)
	}
	if len(capturedReq.Tags) != 2 || capturedReq.Tags[0] != "music" || capturedReq.Tags[1] != "gear" {
		t.Errorf("expected extracted tags [music gear], got %v", capturedReq.Tags)
	}
	if len(capturedReq.Threads) != 1 || capturedReq.Threads[0].ID != "ref_card_001" {
		t.Errorf("expected resolved thread ref_card_001, got %+v", capturedReq.Threads)
	}
	if capturedReq.ExtraMeta["sourceHint"] != "manual" {
		t.Errorf("expected ExtraMeta passed through, got %#v", capturedReq.ExtraMeta)
	}

	var capturedEvent bus.EventThoughtCapturedData
	select {
	case ev := <-evCh:
		var ok bool
		capturedEvent, ok = ev.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("expected EventThoughtCapturedData, got %T", ev.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventThoughtCaptured")
	}

	if capturedEvent.DocID != docID || capturedEvent.Type != "gadget" {
		t.Errorf("unexpected event payload: %+v", capturedEvent)
	}
	if !strings.HasSuffix(capturedEvent.FilePath, ".gadget.md") {
		t.Errorf("expected .gadget.md file, got %s", capturedEvent.FilePath)
	}

	parsed, err := ParseCardFile(capturedEvent.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile failed: %v", err)
	}
	if parsed.Type != CardType("gadget") || parsed.Title != "Pocket Synth" {
		t.Errorf("unexpected parsed card type/title: %q / %q", parsed.Type, parsed.Title)
	}
	if parsed.Meta["price"] != "¥899" || parsed.Meta["sku"] != "OP-1" {
		t.Errorf("expected inlined Meta preserved, got %#v", parsed.Meta)
	}
	if len(parsed.Threads) != 1 || parsed.Threads[0].ID != "ref_card_001" {
		t.Errorf("expected parsed Threads preserved, got %+v", parsed.Threads)
	}

	// 2. Test plugin-managed direct write (res.FilePath != "")
	directMod := &mockCaptureModule{
		name:     "mock_capture_direct",
		cardType: "directcard",
		handler: func(ctx context.Context, req module.CaptureRequest) (*module.CaptureResult, error) {
			p := filepath.Join(scatteringDir, "direct.md")
			_ = os.WriteFile(p, []byte("---\nid: direct_999\ntype: directcard\n---\n\nPlugin wrote this\n"), 0644)
			return &module.CaptureResult{
				DocID:    "direct_999",
				FilePath: p,
				Title:    "Direct",
				Body:     "Plugin wrote this",
			}, nil
		},
	}
	module.Register(directMod)

	directID, err := app.CaptureModuleCard("directcard", "hello", nil, nil)
	if err != nil {
		t.Fatalf("CaptureModuleCard direct failed: %v", err)
	}
	if directID != "direct_999" {
		t.Errorf("expected direct_999, got %s", directID)
	}
	select {
	case ev := <-evCh:
		d, ok := ev.Data.(bus.EventThoughtCapturedData)
		if !ok || d.DocID != "direct_999" || d.Type != "directcard" {
			t.Errorf("unexpected direct capture event: %#v", ev.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for direct EventThoughtCaptured")
	}
}

func TestApp_ReloadExternalPlugins_Command_And_Watcher(t *testing.T) {
	tmpHome := t.TempDir()
	pluginsDir := filepath.Join(tmpHome, ".finnegan", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		module.ResetExternalPlugins()
	})

	app := &App{Home: tmpHome}

	// Verify system:reload_plugins command is registered
	cmds := app.GetRegisteredCommands()
	hasReloadCmd := false
	for _, c := range cmds {
		if c.ID == "system:reload_plugins" && c.Trigger == "reload" && c.Namespace == "system" && c.IsPrimary {
			hasReloadCmd = true
			break
		}
	}
	if !hasReloadCmd {
		t.Fatalf("expected system:reload_plugins command in GetRegisteredCommands, got %+v", cmds)
	}

	writePlugin := func(name string) string {
		script := `#!/bin/sh
if [ "$1" = "--finnegan-manifest" ]; then
  echo '{"name":"` + name + `","themeColor":"#8b5cf6","cardTypes":[{"type":"` + name + `_card"}],"commands":[{"id":"` + name + `:ping","name":"Ping","trigger":"` + name + `"}]}'
  exit 0
fi
read input_json
echo '{"status":"ok"}'
`
		p := filepath.Join(pluginsDir, name+".plugin")
		if err := os.WriteFile(p, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
		return p
	}

	hasManifest := func(manifests []module.ModuleManifest, name string) bool {
		for _, m := range manifests {
			if m.Name == name {
				return true
			}
		}
		return false
	}

	// 1. Add plugin and call ReloadExternalPlugins directly
	p1 := writePlugin("hotplug1")
	manifests, err := app.ReloadExternalPlugins()
	if err != nil {
		t.Fatalf("ReloadExternalPlugins failed: %v", err)
	}
	if !hasManifest(manifests, "hotplug1") {
		t.Fatalf("expected hotplug1 in reloaded manifests, got %+v", manifests)
	}

	// 2. Remove plugin and execute system:reload_plugins (/reload)
	if err := os.Remove(p1); err != nil {
		t.Fatal(err)
	}
	res, err := app.ExecuteModuleCommand("system:reload_plugins", nil)
	if err != nil {
		t.Fatalf("ExecuteModuleCommand(system:reload_plugins) failed: %v", err)
	}
	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["status"] != "reloaded" {
		t.Fatalf("unexpected /reload result: %#v", res)
	}
	if hasManifest(app.GetModuleManifests(), "hotplug1") {
		t.Fatal("expected hotplug1 to be unregistered after removal and /reload")
	}

	// 3. Test debounced fsnotify plugin watcher
	app.startPluginWatcher(pluginsDir)
	defer app.stopPluginWatcher()

	_ = writePlugin("hotplug_watch")
	deadline := time.Now().Add(3 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		if hasManifest(app.GetModuleManifests(), "hotplug_watch") {
			found = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !found {
		t.Fatal("expected debounced fsnotify plugin watcher to load hotplug_watch automatically")
	}
}

