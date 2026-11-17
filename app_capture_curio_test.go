package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"Finnegan/core/bus"
	"Finnegan/core/module"
)

func buildCurioPluginForTest(t *testing.T, outPath string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", outPath, "./plugins/curio")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build curio.plugin: %v\noutput:\n%s", err, string(out))
	}
	if err := os.Chmod(outPath, 0755); err != nil {
		t.Fatalf("failed to chmod curio.plugin: %v", err)
	}
}

func TestCurioDecoupling_UninstalledReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GEMINI_API_KEY", "")

	module.ResetExternalPlugins()
	t.Cleanup(func() {
		module.ResetExternalPlugins()
	})

	if _, exists := module.GetRegistry()["curio"]; exists {
		t.Fatal("expected built-in curio module to be completely purged from Host registry")
	}

	app := &App{
		Home: t.TempDir(),
	}

	if _, err := app.CaptureModuleCard("curio", "Braun ET66 Calculator", nil, nil); err == nil || !strings.Contains(err.Error(), "no CaptureProvider") {
		t.Fatalf("expected no CaptureProvider error when curio.plugin is uninstalled, got: %v", err)
	}

	if _, err := app.CaptureCurio("Braun ET66 Calculator", nil); err == nil || !strings.Contains(err.Error(), "no CaptureProvider") {
		t.Fatalf("expected CaptureCurio to fail with no CaptureProvider when curio.plugin is uninstalled, got: %v", err)
	}

	if _, err := app.CaptureCurio("", []ImagePayload{}); err == nil {
		t.Fatal("expected error with empty text and empty images")
	}
}

func TestCurioExternalPlugin_ClosedLoop(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GEMINI_API_KEY", "")

	module.ResetExternalPlugins()
	t.Cleanup(func() {
		module.ResetExternalPlugins()
	})

	tmpHome := t.TempDir()
	pluginsDir := filepath.Join(tmpHome, ".finnegan", "plugins")
	scatteringDir := filepath.Join(tmpHome, "Scattering")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scatteringDir, 0755); err != nil {
		t.Fatal(err)
	}

	pluginBin := filepath.Join(pluginsDir, "curio.plugin")
	buildCurioPluginForTest(t, pluginBin)

	// 1. Load via LoadExternalPlugins and verify manifest & UI bundle serving
	loaded, err := module.LoadExternalPlugins(pluginsDir)
	if err != nil {
		t.Fatalf("LoadExternalPlugins failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Name() != "curio" {
		t.Fatalf("expected 1 loaded plugin 'curio', got %+v", loaded)
	}

	app := &App{
		Home: tmpHome,
	}

	// ReloadExternalPlugins exercises ResetExternalPlugins + LoadExternalPlugins + Init
	manifests, err := app.ReloadExternalPlugins()
	if err != nil {
		t.Fatalf("ReloadExternalPlugins failed: %v", err)
	}

	var foundCurioManifest *module.ModuleManifest
	for i := range manifests {
		if manifests[i].Name == "curio" {
			foundCurioManifest = &manifests[i]
			break
		}
	}
	if foundCurioManifest == nil {
		t.Fatalf("expected curio in manifests after ReloadExternalPlugins, got %+v", manifests)
	}
	if !foundCurioManifest.IsExternal || foundCurioManifest.ThemeColor != "#10b981" {
		t.Errorf("unexpected curio manifest metadata: %+v", *foundCurioManifest)
	}
	if len(foundCurioManifest.CardTypes) != 1 || foundCurioManifest.CardTypes[0].UIEntry != "/api/plugins/curio/ui.js" {
		t.Errorf("unexpected curio CardTypes: %+v", foundCurioManifest.CardTypes)
	}

	// Verify /api/plugins/curio/ui.js HTTP endpoint serves embedded ui.js
	req := httptest.NewRequest(http.MethodGet, "/api/plugins/curio/ui.js", nil)
	rec := httptest.NewRecorder()
	createAssetHandler(app).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/plugins/curio/ui.js, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "finnegan-plugin-curio-styles") {
		t.Errorf("expected ui.js response to contain curio styles, got:\n%s", rec.Body.String())
	}

	eventChan := bus.GetBus().Subscribe(bus.EventThoughtCaptured, "test_curio_closed_loop")
	defer bus.GetBus().Unsubscribe(bus.EventThoughtCaptured, "test_curio_closed_loop")

	// 2. Text-only capture via CaptureModuleCard
	text := "My New Curio Request #v1.0 #wishlist\nI want to buy this item."
	docID, err := app.CaptureModuleCard("curio", text, nil, nil)
	if err != nil {
		t.Fatalf("CaptureModuleCard(curio) text-only failed: %v", err)
	}
	if docID == "" {
		t.Fatal("expected non-empty docID")
	}

	var textEvt bus.EventThoughtCapturedData
	select {
	case evt := <-eventChan:
		var ok bool
		textEvt, ok = evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("expected EventThoughtCapturedData, got %T", evt.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for text-only EventThoughtCaptured")
	}

	if textEvt.Type != "curio" || textEvt.Title != "My New Curio Request" {
		t.Errorf("unexpected text event: %+v", textEvt)
	}
	if !strings.HasSuffix(textEvt.FilePath, ".curio.md") {
		t.Errorf("expected .curio.md file path, got %q", textEvt.FilePath)
	}

	parsedText, err := ParseCardFile(textEvt.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile(text) failed: %v", err)
	}
	if parsedText.Type != CardTypeCurio || parsedText.Title != "My New Curio Request" {
		t.Errorf("unexpected parsed card type/title: %q / %q", parsedText.Type, parsedText.Title)
	}
	if parsedText.Meta == nil || parsedText.Meta["intent"] != "wish" || parsedText.Meta["availability"] != "available" {
		t.Errorf("expected ParseCardFile to populate Card.Meta intent/availability, got %#v", parsedText.Meta)
	}

	// 3. Image capture via CaptureModuleCard (offline fallback without API key)
	images := []ImagePayload{
		{
			Data: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
			Ext:  ".png",
		},
	}
	imgDocID, err := app.CaptureModuleCard(":curio", "Leica M3 Rangefinder #camera\nClassic chrome body", images, nil)
	if err != nil {
		t.Fatalf("CaptureModuleCard(:curio) image capture failed: %v", err)
	}
	if imgDocID == "" {
		t.Fatal("expected non-empty imgDocID")
	}

	var imgEvt bus.EventThoughtCapturedData
	select {
	case evt := <-eventChan:
		var ok bool
		imgEvt, ok = evt.Data.(bus.EventThoughtCapturedData)
		if !ok {
			t.Fatalf("expected EventThoughtCapturedData, got %T", evt.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for image EventThoughtCaptured")
	}

	if imgEvt.Type != "curio" || len(imgEvt.Images) != 1 || !strings.HasPrefix(imgEvt.Images[0], "curio/") {
		t.Errorf("unexpected image event payload: %+v", imgEvt)
	}

	parsedImg, err := ParseCardFile(imgEvt.FilePath)
	if err != nil {
		t.Fatalf("ParseCardFile(image) failed: %v", err)
	}
	if parsedImg.Type != CardTypeCurio || parsedImg.Title != "Leica M3 Rangefinder camera" {
		t.Errorf("unexpected parsed image card: type=%q title=%q", parsedImg.Type, parsedImg.Title)
	}
	if parsedImg.Meta == nil {
		t.Fatal("expected non-nil Card.Meta after ParseCardFile")
	}
	if parsedImg.Meta["intent"] != "wish" || parsedImg.Meta["availability"] != "available" {
		t.Errorf("unexpected Card.Meta defaults: %#v", parsedImg.Meta)
	}
	snapPath, _ := parsedImg.Meta["snapshotPath"].(string)
	shotPath, _ := parsedImg.Meta["screenshotPath"].(string)
	if snapPath == "" || shotPath == "" {
		t.Errorf("expected snapshotPath and screenshotPath in Card.Meta, got %#v", parsedImg.Meta)
	}
	if _, err := os.Stat(snapPath); err != nil {
		t.Errorf("expected snapshot file on disk at %s: %v", snapPath, err)
	}
	if _, err := os.Stat(shotPath); err != nil {
		t.Errorf("expected hero image file on disk at %s: %v", shotPath, err)
	}

	// 4. Remove curio.plugin and verify ResetExternalPlugins + ReloadExternalPlugins unregister curio
	if err := os.Remove(pluginBin); err != nil {
		t.Fatal(err)
	}
	reloadedAfterRemove, err := app.ReloadExternalPlugins()
	if err != nil {
		t.Fatalf("ReloadExternalPlugins after remove failed: %v", err)
	}
	for _, m := range reloadedAfterRemove {
		if m.Name == "curio" {
			t.Fatal("expected curio to be unregistered when curio.plugin is removed")
		}
	}
	if _, exists := module.GetRegistry()["curio"]; exists {
		t.Fatal("expected curio absent from module.GetRegistry() after ResetExternalPlugins/ReloadExternalPlugins")
	}
	if _, err := app.CaptureModuleCard("curio", "should fail", nil, nil); err == nil {
		t.Fatal("expected CaptureModuleCard('curio') to return error after curio.plugin removal")
	}
}
