package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	goRuntime "runtime"
	"sort"
	"strings"

	"Finnegan/core/module"
	"Finnegan/core/search"
	"Finnegan/core/watcher"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type WatcherStorage struct {
	ctx        context.Context
	onChange   func(id, data string)
	onDelete   func(id string)
	onConflict func(id string, paths []string)
}

func (w *WatcherStorage) SetOnChange(fn func(id, data string)) {
	w.onChange = fn
}

func (w *WatcherStorage) SetOnDelete(fn func(id string)) {
	w.onDelete = fn
}

func (w *WatcherStorage) SetOnConflict(fn func(id string, paths []string)) {
	w.onConflict = fn
}

func (w *WatcherStorage) SaveNote(id, data string) error {
	if w.onChange != nil {
		w.onChange(id, data)
	} else if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "notes:changed", map[string]interface{}{
			"docId": id,
			"op":    "upsert",
		})
	}
	return nil
}

func (w *WatcherStorage) DeleteNote(id string) error {
	if w.onDelete != nil {
		w.onDelete(id)
	} else if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "notes:changed", map[string]interface{}{
			"docId": id,
			"op":    "remove",
		})
	}
	return nil
}

func (w *WatcherStorage) GetNote(id string) (string, error) {
	return "", nil
}

func (w *WatcherStorage) EmitConflict(id string, paths []string) {
	if w.onConflict != nil {
		w.onConflict(id, paths)
	} else if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "card:id_conflict", map[string]interface{}{
			"id":    id,
			"paths": paths,
		})
	}
}

func (a *App) GetRadarTargets() []interface{} {
	m, ok := module.GetRegistry()["bugu"]
	if !ok || m == nil {
		return []interface{}{}
	}
	rv := reflect.ValueOf(m)
	method := rv.MethodByName("GetRadarTargets")
	if !method.IsValid() {
		return []interface{}{}
	}
	out := method.Call(nil)
	if len(out) == 0 || !out[0].IsValid() {
		return []interface{}{}
	}
	sliceVal := out[0]
	if sliceVal.Kind() != reflect.Slice {
		return []interface{}{}
	}
	res := make([]interface{}, sliceVal.Len())
	for i := 0; i < sliceVal.Len(); i++ {
		res[i] = sliceVal.Index(i).Interface()
	}
	return res
}

func (a *App) RunRadar(intent string) interface{} {
	m, ok := module.GetRegistry()["bugu"]
	if !ok || m == nil {
		return map[string]interface{}{
			"status": "error",
			"error":  "Bugu configuration is not loaded",
		}
	}
	rv := reflect.ValueOf(m)
	method := rv.MethodByName("RunRadar")
	if method.IsValid() {
		out := method.Call([]reflect.Value{reflect.ValueOf(intent)})
		if len(out) > 0 && out[0].IsValid() {
			return out[0].Interface()
		}
	}
	res, err := a.ExecuteModuleCommand("bugu:radar", map[string]interface{}{"text": intent})
	if err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}
	return res
}

func (a *App) GetTempCards() []Card {
	return a.CardManager.GetTempCards()
}

func (a *App) PromoteTempCard(cardID string, comment string, echoOf string) (*Card, error) {
	return a.CardManager.PromoteToLocalWithOpts(cardID, a.Home, comment, echoOf)
}

func (a *App) GetNote(id string) (string, error) {
	return "", nil
}

func (a *App) ClearTempCards() {
	a.CardManager.ClearTempCards()
}

func (a *App) EvictTempCard(cardID string) bool {
	if a.CardManager == nil {
		return false
	}
	return a.CardManager.EvictTempCard(cardID)
}

// findCardPath resolves a cardID or path to a concrete markdown file path.
func (a *App) findCardPath(cardID string) string {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return ""
	}

	// 1. Direct path check if cardID is already a valid file path
	if fi, err := os.Stat(cardID); err == nil && !fi.IsDir() {
		return cardID
	}

	// 2. Lookup via GlobalWatcher
	if w := watcher.GetWatcher(); w != nil {
		if p := w.GetPathForID(cardID); p != "" {
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p
			}
		}
	}

	// 3. Scan Scattering and Weaving directories by filename or frontmatter id
	if a.Home != "" {
		dirs := []string{
			filepath.Join(a.Home, "Scattering"),
			filepath.Join(a.Home, "Weaving"),
		}
		for _, dir := range dirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					candidate := filepath.Join(dir, e.Name())
					if strings.Contains(e.Name(), cardID) {
						return candidate
					}
					if parsed, err := ParseCardFile(candidate); err == nil && parsed != nil {
						if parsed.ID == cardID {
							return candidate
						}
					}
				}
			}
		}
	}

	return ""
}

// OpenCardInEditor opens the specified card's markdown file in an editor.
// Defaults to qled:// protocol scheme; if unavailable or fails, falls back to the system default application.
func (a *App) OpenCardInEditor(cardID string) error {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return fmt.Errorf("empty card id")
	}

	filePath := a.findCardPath(cardID)

	// If card is temporary in memory, promote it to a real file in Scattering
	if filePath == "" && a.CardManager != nil {
		if promoted, err := a.CardManager.PromoteToLocal(cardID, a.Home); err == nil && promoted != nil {
			filePath = promoted.FilePath
		}
	}

	if filePath == "" {
		return fmt.Errorf("could not locate markdown file for card: %s", cardID)
	}

	return openFileWithEditor(filePath)
}

// ReloadCard forces a fresh read of the card from disk and returns the updated Card.
func (a *App) ReloadCard(cardID string) (*Card, error) {
	filePath := a.findCardPath(cardID)
	if filePath == "" {
		return nil, fmt.Errorf("file not found for card: %s", cardID)
	}

	return ParseCardFile(filePath)
}

func openFileWithEditor(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err == nil {
		filePath = absPath
	}

	qledURI := fmt.Sprintf("qled://open?path=%s", url.PathEscape(filePath))

	switch goRuntime.GOOS {
	case "darwin":
		// 1. Try qled protocol
		cmd := exec.Command("open", qledURI)
		if err := cmd.Run(); err == nil {
			return nil
		}
		// 2. Fallback to macOS system default file opener
		fallbackCmd := exec.Command("open", filePath)
		return fallbackCmd.Run()

	case "windows":
		cmd := exec.Command("cmd", "/c", "start", "", qledURI)
		if err := cmd.Run(); err == nil {
			return nil
		}
		fallbackCmd := exec.Command("cmd", "/c", "start", "", filePath)
		return fallbackCmd.Run()

	default: // Linux and others
		cmd := exec.Command("xdg-open", qledURI)
		if err := cmd.Run(); err == nil {
			return nil
		}
		fallbackCmd := exec.Command("xdg-open", filePath)
		return fallbackCmd.Run()
	}
}

// SearchLocalCards loads local cards and temporary cards, runs fuzzy search, and returns sorted results.
func (a *App) SearchLocalCards(query string) []Card {
	var rawCards []Card
	rawCards = append(rawCards, a.LoadLocalCards()...)
	rawCards = append(rawCards, a.GetTempCards()...)

	// Deduplicate rawCards by ID and content
	seenID := make(map[string]bool)
	seenContent := make(map[string]bool)
	var uniqueCards []Card
	for _, c := range rawCards {
		trimmedContent := strings.TrimSpace(c.Content)
		if c.ID != "" && seenID[c.ID] {
			continue
		}
		if trimmedContent != "" && seenContent[trimmedContent] {
			continue
		}
		if c.ID != "" {
			seenID[c.ID] = true
		}
		if trimmedContent != "" {
			seenContent[trimmedContent] = true
		}
		uniqueCards = append(uniqueCards, c)
	}

	sCards := make([]search.Card, len(uniqueCards))
	for i, c := range uniqueCards {
		sCards[i] = search.Card{
			ID:             c.ID,
			Type:           string(c.Type),
			Title:          c.Title,
			Content:        c.Content,
			Tags:           c.Tags,
			Keywords:       c.Keywords,
			SourceURL:      c.SourceURL,
			OriginalText:   c.OriginalText,
			TranslatedText: c.TranslatedText,
			EchoOf:         c.EchoOf,
			CreatedAt:      c.CreatedAt,
		}
	}

	matched := search.SearchCards(query, sCards)

	cardMap := make(map[string]Card, len(uniqueCards))
	for _, c := range uniqueCards {
		cardMap[c.ID] = c
	}

	resSeenID := make(map[string]bool)
	resSeenContent := make(map[string]bool)
	results := make([]Card, 0, len(matched))
	for _, sc := range matched {
		trimmedContent := strings.TrimSpace(sc.Content)
		if sc.ID != "" && resSeenID[sc.ID] {
			continue
		}
		if trimmedContent != "" && resSeenContent[trimmedContent] {
			continue
		}
		if sc.ID != "" {
			resSeenID[sc.ID] = true
		}
		if trimmedContent != "" {
			resSeenContent[trimmedContent] = true
		}

		if orig, ok := cardMap[sc.ID]; ok {
			results = append(results, orig)
		} else {
			results = append(results, Card{
				ID:             sc.ID,
				Type:           CardType(sc.Type),
				Title:          sc.Title,
				Content:        sc.Content,
				SourceURL:      sc.SourceURL,
				OriginalText:   sc.OriginalText,
				TranslatedText: sc.TranslatedText,
				EchoOf:         sc.EchoOf,
				CreatedAt:      sc.CreatedAt,
			})
		}
	}

	return results
}

// GetRegisteredCommands iterates module.GetRegistry, finds modules implementing CommandProvider, and collects all CommandDefs.
func (a *App) GetRegisteredCommands() []module.CommandDef {
	commands := make([]module.CommandDef, 0)
	for _, m := range module.GetRegistry() {
		if cp, ok := m.(module.CommandProvider); ok {
			commands = append(commands, cp.GetCommands()...)
		}
	}

	return commands
}

// GetModuleManifests returns the normalized ModuleManifest for all registered built-in and external modules.
func (a *App) GetModuleManifests() []module.ModuleManifest {
	reg := module.GetRegistry()
	manifests := make([]module.ModuleManifest, 0, len(reg))
	for _, m := range reg {
		manifests = append(manifests, module.BuildModuleManifest(m))
	}
	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].Name < manifests[j].Name
	})
	return manifests
}

// ExecuteModuleAction dispatches a custom UI/RPC action to the target module's ActionProvider.
func (a *App) ExecuteModuleAction(moduleName string, actionID string, payload map[string]interface{}) (interface{}, error) {
	m, ok := module.GetRegistry()[moduleName]
	if !ok || m == nil {
		return nil, fmt.Errorf("module not found: %s", moduleName)
	}
	ap, ok := m.(module.ActionProvider)
	if !ok {
		return nil, fmt.Errorf("module %s does not implement ActionProvider", moduleName)
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return ap.ExecuteAction(ctx, actionID, payload)
}

// ExecuteModuleCommand dispatches a command to the module that owns cmdID.
func (a *App) ExecuteModuleCommand(cmdID string, params map[string]interface{}) (interface{}, error) {
	if cmdID == "core:note" {
		text := ""
		if t, ok := params["text"].(string); ok {
			text = t
		} else if c, ok := params["content"].(string); ok {
			text = c
		}
		docID, err := a.CaptureNote(text)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status": "success",
			"docID":  docID,
		}, nil
	}

	for _, m := range module.GetRegistry() {
		if cp, ok := m.(module.CommandProvider); ok {
			for _, cmd := range cp.GetCommands() {
				if cmd.ID == cmdID {
					ctx := a.ctx
					if ctx == nil {
						ctx = context.Background()
					}
					return cp.ExecuteCommand(ctx, cmdID, params)
				}
			}
		}
	}
	return nil, fmt.Errorf("command not found: %s", cmdID)
}
