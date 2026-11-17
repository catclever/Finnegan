package module

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"Finnegan/core/models"
	"gopkg.in/yaml.v3"
)

// ExternalModule wraps an external out-of-process plugin executable (either a directory plugin or a standalone .plugin binary)
// and implements Module, CommandProvider, ManifestProvider, ActionProvider, and CaptureProvider.
type ExternalModule struct {
	manifest  ModuleManifest
	execPath  string
	pluginDir string
	uiFile    string
	homeDir   string
	ctx       *Context
	rawConfig map[string]interface{}
}

var (
	_ Module           = (*ExternalModule)(nil)
	_ CommandProvider  = (*ExternalModule)(nil)
	_ ManifestProvider = (*ExternalModule)(nil)
	_ ActionProvider   = (*ExternalModule)(nil)
	_ CaptureProvider  = (*ExternalModule)(nil)
)

func (e *ExternalModule) Name() string {
	return e.manifest.Name
}

func (e *ExternalModule) ConfigTarget() interface{} {
	if e.rawConfig == nil {
		e.rawConfig = make(map[string]interface{})
	}
	return &e.rawConfig
}

func (e *ExternalModule) Init(ctx *Context) error {
	e.ctx = ctx
	return nil
}

func (e *ExternalModule) Bindings() []interface{} {
	return nil
}

func (e *ExternalModule) GetManifest() ModuleManifest {
	return e.manifest
}

func (e *ExternalModule) GetCommands() []CommandDef {
	return e.manifest.Commands
}

func (e *ExternalModule) ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error) {
	return e.invokeSubprocess(ctx, "execute_command", cmdID, params)
}

func (e *ExternalModule) ExecuteAction(ctx context.Context, actionID string, payload map[string]interface{}) (interface{}, error) {
	return e.invokeSubprocess(ctx, "execute_action", actionID, payload)
}

// CaptureCard delegates card capture to the external plugin subprocess using method "capture_card".
func (e *ExternalModule) CaptureCard(ctx context.Context, req CaptureRequest) (*CaptureResult, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capture request: %w", err)
	}
	var params map[string]interface{}
	if err := json.Unmarshal(reqBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to decode capture params: %w", err)
	}

	res, err := e.invokeSubprocess(ctx, "capture_card", req.CardType, params)
	if err != nil {
		return nil, err
	}

	resMap, ok := res.(map[string]interface{})
	if !ok {
		return &CaptureResult{}, nil
	}
	if errMsg, hasErr := resMap["error"].(string); hasErr && errMsg != "" {
		return nil, fmt.Errorf("plugin %s capture_card error: %s", e.manifest.Name, errMsg)
	}

	targetMap := resMap
	if inner, ok := resMap["result"].(map[string]interface{}); ok {
		targetMap = inner
	}

	b, err := json.Marshal(targetMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capture result: %w", err)
	}
	var capRes CaptureResult
	if err := json.Unmarshal(b, &capRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal capture result: %w", err)
	}
	if capRes.DocID == "" {
		if v, ok := targetMap["doc_id"].(string); ok {
			capRes.DocID = v
		}
	}
	if capRes.FilePath == "" {
		if v, ok := targetMap["file_path"].(string); ok {
			capRes.FilePath = v
		}
	}
	if capRes.FileExt == "" {
		if v, ok := targetMap["file_ext"].(string); ok {
			capRes.FileExt = v
		}
	}
	return &capRes, nil
}

func (e *ExternalModule) resolveHomeDir() string {
	if e.ctx != nil && e.ctx.HomeDir != "" {
		return e.ctx.HomeDir
	}
	if e.homeDir != "" {
		return e.homeDir
	}
	return deriveHomeFromPluginsDir(filepath.Dir(e.execPath))
}

func deriveHomeFromPluginsDir(pluginsDir string) string {
	clean := filepath.Clean(pluginsDir)
	if filepath.Base(clean) == "plugins" && filepath.Base(filepath.Dir(clean)) == ".finnegan" {
		return filepath.Dir(filepath.Dir(clean))
	}
	if filepath.Base(filepath.Dir(clean)) == "plugins" && filepath.Base(filepath.Dir(filepath.Dir(clean))) == ".finnegan" {
		return filepath.Dir(filepath.Dir(filepath.Dir(clean)))
	}
	return ""
}

func buildPluginEnv(homeDir, pluginName string) []string {
	env := os.Environ()
	if homeDir == "" {
		return env
	}
	filtered := make([]string, 0, len(env)+2)
	for _, kv := range env {
		if strings.HasPrefix(kv, "FINNEGAN_HOME=") || strings.HasPrefix(kv, "FINNEGAN_PLUGIN_DATA_DIR=") {
			continue
		}
		filtered = append(filtered, kv)
	}
	filtered = append(filtered, "FINNEGAN_HOME="+homeDir)
	if pluginName != "" {
		dataDir := filepath.Join(homeDir, ".finnegan", "data", pluginName)
		_ = os.MkdirAll(dataDir, 0755)
		filtered = append(filtered, "FINNEGAN_PLUGIN_DATA_DIR="+dataDir)
	}
	return filtered
}

// GetUIBundle returns the JavaScript ESM bundle for this external plugin's custom card UI.
func (e *ExternalModule) GetUIBundle() ([]byte, error) {
	if e.uiFile != "" {
		if data, err := os.ReadFile(e.uiFile); err == nil {
			return data, nil
		}
	}
	if e.execPath == "" {
		return nil, fmt.Errorf("no UI bundle or executable configured for plugin %s", e.manifest.Name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.execPath, "--finnegan-ui")
	if e.pluginDir != "" {
		cmd.Dir = e.pluginDir
	}
	cmd.Env = buildPluginEnv(e.resolveHomeDir(), e.manifest.Name)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to extract UI bundle from plugin %s: %w", e.manifest.Name, err)
	}
	return out, nil
}

func (e *ExternalModule) invokeSubprocess(ctx context.Context, method string, id string, params map[string]interface{}) (interface{}, error) {
	if e.execPath == "" {
		return nil, fmt.Errorf("external plugin %s has no executable path", e.manifest.Name)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var llms map[string]LLMConfig
	if e.ctx != nil {
		llms = e.ctx.LLMs
	}

	reqPayload := map[string]interface{}{
		"method": method,
		"id":     id,
		"params": params,
		"llms":   llms,
		"config": e.rawConfig,
	}
	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin request: %w", err)
	}

	cmd := exec.CommandContext(ctx, e.execPath)
	if e.pluginDir != "" {
		cmd.Dir = e.pluginDir
	}
	cmd.Env = buildPluginEnv(e.resolveHomeDir(), e.manifest.Name)
	cmd.Stdin = bytes.NewReader(append(reqBytes, '\n'))

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to attach stdout pipe: %w", err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start plugin %s: %w", e.manifest.Name, err)
	}

	var finalResult interface{}
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if evt, ok := msg["event"].(string); ok && evt != "" {
			e.handleStreamEvent(evt, msg)
			continue
		}

		finalResult = msg
	}

	if err := cmd.Wait(); err != nil {
		errText := strings.TrimSpace(stderrBuf.String())
		if errText != "" {
			return nil, fmt.Errorf("plugin %s execution failed: %w (%s)", e.manifest.Name, err, errText)
		}
		return nil, fmt.Errorf("plugin %s execution failed: %w", e.manifest.Name, err)
	}

	if finalResult == nil {
		finalResult = map[string]interface{}{
			"status": "ok",
		}
	}
	return finalResult, nil
}

func (e *ExternalModule) handleStreamEvent(evt string, msg map[string]interface{}) {
	if e.ctx == nil || e.ctx.Host == nil {
		return
	}

	switch evt {
	case "spawn_card":
		rawCard, ok := msg["card"]
		if !ok || rawCard == nil {
			return
		}
		cardBytes, err := json.Marshal(rawCard)
		if err != nil {
			return
		}
		var card models.BabelCard
		if err := json.Unmarshal(cardBytes, &card); err != nil {
			return
		}
		// Support camelCase aliases from plugin JSON output as well
		if cardMap, ok := rawCard.(map[string]interface{}); ok {
			if card.SourceURL == "" {
				if v, ok := cardMap["sourceUrl"].(string); ok {
					card.SourceURL = v
				}
			}
			if card.OriginalText == "" {
				if v, ok := cardMap["originalText"].(string); ok {
					card.OriginalText = v
				}
			}
			if card.TranslatedText == "" {
				if v, ok := cardMap["translatedText"].(string); ok {
					card.TranslatedText = v
				}
			}
			if card.PublishedAt == "" {
				if v, ok := cardMap["publishedAt"].(string); ok {
					card.PublishedAt = v
				}
			}
			if card.AuthorName == "" {
				if v, ok := cardMap["authorName"].(string); ok {
					card.AuthorName = v
				}
			}
			if card.AuthorAvatar == "" {
				if v, ok := cardMap["authorAvatar"].(string); ok {
					card.AuthorAvatar = v
				}
			}
			if card.EchoOf == "" {
				if v, ok := cardMap["echoOf"].(string); ok {
					card.EchoOf = v
				}
			}
			if card.MatchedAnchor == "" {
				if v, ok := cardMap["matchedAnchor"].(string); ok {
					card.MatchedAnchor = v
				}
			}
			if card.EchoReason == "" {
				if v, ok := cardMap["echoReason"].(string); ok {
					card.EchoReason = v
				}
			}
		}
		if card.Origin == "" {
			card.Origin = e.manifest.Name
		}
		e.ctx.Host.SpawnCard(card)

	case "complete_session":
		cardID, _ := msg["cardId"].(string)
		if cardID == "" {
			cardID, _ = msg["card_id"].(string)
		}
		e.ctx.Host.CompleteSession(cardID)

	default:
		e.ctx.Host.EmitEvent(evt, msg["data"])
	}
}

// LoadExternalPlugins scans pluginsDir for directory-based plugins (<dir>/plugin.yml)
// and standalone single-file plugins (<name>.plugin), registers them in the global registry, and returns them.
func LoadExternalPlugins(pluginsDir string) ([]Module, error) {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	homeDir := deriveHomeFromPluginsDir(pluginsDir)
	if homeDir == "" {
		homeDir = os.Getenv("FINNEGAN_HOME")
	}

	var loaded []Module
	for _, entry := range entries {
		fullPath := filepath.Join(pluginsDir, entry.Name())
		if entry.IsDir() {
			extMod, err := loadDirectoryPlugin(fullPath, homeDir)
			if err != nil || extMod == nil {
				continue
			}
			if homeDir != "" {
				_ = os.MkdirAll(filepath.Join(homeDir, ".finnegan", "data", extMod.manifest.Name), 0755)
			}
			Register(extMod)
			loaded = append(loaded, extMod)
		} else if strings.HasSuffix(entry.Name(), ".plugin") {
			extMod, err := loadSingleBinaryPlugin(fullPath, homeDir)
			if err != nil || extMod == nil {
				continue
			}
			if homeDir != "" {
				_ = os.MkdirAll(filepath.Join(homeDir, ".finnegan", "data", extMod.manifest.Name), 0755)
			}
			Register(extMod)
			loaded = append(loaded, extMod)
		}
	}

	return loaded, nil
}

func loadDirectoryPlugin(dirPath string, homeDir string) (*ExternalModule, error) {
	manifestCandidates := []string{
		filepath.Join(dirPath, "plugin.yml"),
		filepath.Join(dirPath, "plugin.yaml"),
		filepath.Join(dirPath, "manifest.json"),
	}
	var manifestBytes []byte
	var isJSON bool
	for _, p := range manifestCandidates {
		if data, err := os.ReadFile(p); err == nil {
			manifestBytes = data
			isJSON = strings.HasSuffix(p, ".json")
			break
		}
	}
	if len(manifestBytes) == 0 {
		return nil, fmt.Errorf("no plugin manifest found in %s", dirPath)
	}

	var manifest ModuleManifest
	if isJSON {
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
			return nil, err
		}
	}

	if manifest.Name == "" {
		manifest.Name = filepath.Base(dirPath)
	}
	manifest.IsExternal = true

	entryBin := manifest.Entry
	if entryBin == "" {
		entryBin = manifest.Name
	}
	execPath := filepath.Join(dirPath, entryBin)

	uiFile := filepath.Join(dirPath, "ui.js")
	hasUIFile := false
	if fi, err := os.Stat(uiFile); err == nil && !fi.IsDir() {
		hasUIFile = true
	} else {
		uiFile = ""
	}

	if hasUIFile {
		defaultUIEntry := fmt.Sprintf("/api/plugins/%s/ui.js", manifest.Name)
		for i := range manifest.CardTypes {
			if manifest.CardTypes[i].UIEntry == "" {
				manifest.CardTypes[i].UIEntry = defaultUIEntry
			}
		}
	}

	return &ExternalModule{
		manifest:  manifest,
		execPath:  execPath,
		pluginDir: dirPath,
		uiFile:    uiFile,
		homeDir:   homeDir,
	}, nil
}

func loadSingleBinaryPlugin(binPath string, homeDir string) (*ExternalModule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initialName := strings.TrimSuffix(filepath.Base(binPath), ".plugin")
	cmd := exec.CommandContext(ctx, binPath, "--finnegan-manifest")
	cmd.Dir = filepath.Dir(binPath)
	cmd.Env = buildPluginEnv(homeDir, initialName)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query --finnegan-manifest from %s: %w", binPath, err)
	}

	var manifest ModuleManifest
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if err := json.Unmarshal(trimmed, &manifest); err != nil {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(trimmed, &manifest); err != nil {
			return nil, err
		}
	}

	if manifest.Name == "" {
		manifest.Name = initialName
	}
	manifest.IsExternal = true

	return &ExternalModule{
		manifest:  manifest,
		execPath:  binPath,
		pluginDir: filepath.Dir(binPath),
		homeDir:   homeDir,
	}, nil
}
