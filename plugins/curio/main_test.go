package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const samplePNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

func TestCurioPlugin_ManifestFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"curio.plugin", "--finnegan-manifest"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	var m pluginManifest
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatalf("failed to decode manifest JSON: %v\nraw=%s", err, stdout.String())
	}
	if m.Name != "curio" {
		t.Errorf("expected name 'curio', got %q", m.Name)
	}
	if m.ThemeColor != "#10b981" {
		t.Errorf("expected themeColor '#10b981', got %q", m.ThemeColor)
	}
	if len(m.CardTypes) != 1 {
		t.Fatalf("expected 1 cardType, got %d", len(m.CardTypes))
	}
	ct := m.CardTypes[0]
	if ct.Type != "curio" || ct.Prefix != ":curio" || !ct.Selectable || ct.UIEntry != "/api/plugins/curio/ui.js" {
		t.Errorf("unexpected cardType spec: %+v", ct)
	}
	if len(m.Commands) != 1 || m.Commands[0].ID != "curio:ingest" {
		t.Errorf("unexpected commands: %+v", m.Commands)
	}
}

func TestCurioPlugin_UIFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"curio.plugin", "--finnegan-ui"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	ui := stdout.String()
	for _, needle := range []string{
		"finnegan-plugin-curio-styles",
		"export function mount",
		"export default { mount }",
		"resolveAssetUrl",
		"emitTagClick",
		"openURL",
		"snapshotPath",
		"screenshotPath",
	} {
		if !strings.Contains(ui, needle) {
			t.Errorf("expected embedded ui.js to contain %q", needle)
		}
	}
}

func TestCurioPlugin_CaptureCard_TextOnly(t *testing.T) {
	reqJSON := `{
		"method": "capture_card",
		"id": "curio",
		"params": {
			"cardType": "curio",
			"text": "Braun ET66 Calculator #bauhaus #design\nIconic Dieter Rams design.",
			"cleanText": "Braun ET66 Calculator\nIconic Dieter Rams design.",
			"tags": ["bauhaus", "design"]
		}
	}`

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"curio.plugin"}, strings.NewReader(reqJSON), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	var resp struct {
		Result captureResult `json:"result"`
		Error  string        `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v\nraw=%s", err, stdout.String())
	}
	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	if resp.Result.DocID == "" {
		t.Error("expected non-empty docId")
	}
	if resp.Result.FileExt != ".curio.md" {
		t.Errorf("expected fileExt '.curio.md', got %q", resp.Result.FileExt)
	}
	if resp.Result.Title != "Braun ET66 Calculator" {
		t.Errorf("expected title 'Braun ET66 Calculator', got %q", resp.Result.Title)
	}
	if resp.Result.Meta["intent"] != "wish" || resp.Result.Meta["availability"] != "available" {
		t.Errorf("unexpected meta: %+v", resp.Result.Meta)
	}
	if len(resp.Result.Tags) != 2 {
		t.Errorf("expected 2 tags, got %v", resp.Result.Tags)
	}
}

func TestCurioPlugin_CaptureCard_WithImage_OfflineFallback(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	tmpHome := t.TempDir()

	reqObj := map[string]interface{}{
		"method": "capture_card",
		"id":     "curio",
		"params": map[string]interface{}{
			"cardType":  "curio",
			"text":      "Leica M3 Chrome #camera\nClassic rangefinder",
			"cleanText": "Leica M3 Chrome\nClassic rangefinder",
			"tags":      []string{"camera"},
			"homeDir":   tmpHome,
			"images": []map[string]string{
				{
					"data": "data:image/png;base64," + samplePNGBase64,
					"ext":  ".png",
				},
			},
		},
	}
	reqBytes, _ := json.Marshal(reqObj)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"curio.plugin"}, bytes.NewReader(reqBytes), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	var resp struct {
		Result captureResult `json:"result"`
		Error  string        `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v\nraw=%s", err, stdout.String())
	}
	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	if resp.Result.FileExt != ".curio.md" {
		t.Errorf("expected fileExt '.curio.md', got %q", resp.Result.FileExt)
	}
	if resp.Result.Title != "Leica M3 Chrome" {
		t.Errorf("expected derived title 'Leica M3 Chrome', got %q", resp.Result.Title)
	}
	if len(resp.Result.Images) != 1 || !strings.HasPrefix(resp.Result.Images[0], "curio/") {
		t.Errorf("expected relative curio/ hero image, got %v", resp.Result.Images)
	}
	snapshotPath, _ := resp.Result.Meta["snapshotPath"].(string)
	screenshotPath, _ := resp.Result.Meta["screenshotPath"].(string)
	if snapshotPath == "" || !strings.HasPrefix(snapshotPath, filepath.Join(tmpHome, "archives")) {
		t.Errorf("expected snapshotPath in archives, got %q", snapshotPath)
	}
	if screenshotPath == "" || !strings.HasPrefix(screenshotPath, filepath.Join(tmpHome, "assets", "curio")) {
		t.Errorf("expected screenshotPath in assets/curio, got %q", screenshotPath)
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Errorf("expected snapshot file on disk at %s: %v", snapshotPath, err)
	}
	if _, err := os.Stat(screenshotPath); err != nil {
		t.Errorf("expected hero image file on disk at %s: %v", screenshotPath, err)
	}
	if resp.Result.Meta["intent"] != "wish" || resp.Result.Meta["availability"] != "available" {
		t.Errorf("unexpected meta defaults: %+v", resp.Result.Meta)
	}
}

func TestCurioPlugin_ExecuteCommand_Ingest(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	tmpHome := t.TempDir()
	t.Setenv("FINNEGAN_HOME", tmpHome)

	rawPNG, err := base64.StdEncoding.DecodeString(samplePNGBase64)
	if err != nil {
		t.Fatal(err)
	}
	srcImg := filepath.Join(tmpHome, "input.png")
	if err := os.WriteFile(srcImg, rawPNG, 0644); err != nil {
		t.Fatal(err)
	}

	reqObj := map[string]interface{}{
		"method": "execute_command",
		"id":     "curio:ingest",
		"params": map[string]interface{}{
			"screenshot": srcImg,
			"note":       "Imported via command",
		},
	}
	reqBytes, _ := json.Marshal(reqObj)

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"curio.plugin"}, bytes.NewReader(reqBytes), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	var resp struct {
		Result map[string]interface{} `json:"result"`
		Error  string                 `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode command response: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	filePath, _ := resp.Result["filePath"].(string)
	if filePath == "" || !strings.HasSuffix(filePath, ".curio.md") {
		t.Fatalf("expected .curio.md filePath, got %q", filePath)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read created .curio.md: %v", err)
	}
	if !strings.Contains(string(content), "type: curio") {
		t.Errorf("expected 'type: curio' in file:\n%s", string(content))
	}
}
