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

