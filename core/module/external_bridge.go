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
// and implements Module, CommandProvider, ManifestProvider, and ActionProvider.
type ExternalModule struct {
	manifest  ModuleManifest
	execPath  string
	pluginDir string
	uiFile    string
	ctx       *Context
	rawConfig map[string]interface{}
}

var (
	_ Module           = (*ExternalModule)(nil)
	_ CommandProvider  = (*ExternalModule)(nil)
	_ ManifestProvider = (*ExternalModule)(nil)
	_ ActionProvider   = (*ExternalModule)(nil)
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

	var loaded []Module
	for _, entry := range entries {
		fullPath := filepath.Join(pluginsDir, entry.Name())
		if entry.IsDir() {
			extMod, err := loadDirectoryPlugin(fullPath)
			if err != nil || extMod == nil {
				continue
			}
			Register(extMod)
			loaded = append(loaded, extMod)
		} else if strings.HasSuffix(entry.Name(), ".plugin") {
			extMod, err := loadSingleBinaryPlugin(fullPath)
			if err != nil || extMod == nil {
				continue
			}
			Register(extMod)
			loaded = append(loaded, extMod)
		}
	}

	return loaded, nil
}

func loadDirectoryPlugin(dirPath string) (*ExternalModule, error) {
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
	}, nil
}

func loadSingleBinaryPlugin(binPath string) (*ExternalModule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "--finnegan-manifest")
	cmd.Dir = filepath.Dir(binPath)
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
		base := filepath.Base(binPath)
		manifest.Name = strings.TrimSuffix(base, ".plugin")
	}
	manifest.IsExternal = true

	return &ExternalModule{
		manifest:  manifest,
		execPath:  binPath,
		pluginDir: filepath.Dir(binPath),
	}, nil
}
