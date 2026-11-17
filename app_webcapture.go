package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Finnegan/core/bus"
	"Finnegan/core/config"
	"Finnegan/core/watcher"
	"Finnegan/core/webcapture"
)

// capturePipelineFactory allows injecting a mock or configured pipeline during tests.
var capturePipelineFactory = func(cfg config.JinaConfig) *webcapture.Pipeline {
	return webcapture.NewPipeline(cfg)
}

// getJinaConfig retrieves the active Jina configuration from global config or KeywordEngine,
// reloading from config.LoadConfig() when APIKey is not cached and checking legacy bugu.jina_api_key.
func (a *App) getJinaConfig() config.JinaConfig {
	var res config.JinaConfig
	cfg, _, err := config.LoadConfig()
	if err == nil && cfg != nil {
		res = cfg.Jina
		if res.APIKey == "" {
			if node, exists := cfg.RawModules["bugu"]; exists {
				var buguMeta struct {
					JinaAPIKey string `yaml:"jina_api_key"`
				}
				if err := node.Decode(&buguMeta); err == nil && buguMeta.JinaAPIKey != "" {
					res.APIKey = buguMeta.JinaAPIKey
				}
			}
		}
	}
	if a != nil && a.KeywordEngine != nil {
		if res.APIKey == "" && a.KeywordEngine.JinaConfig.APIKey != "" {
			res.APIKey = a.KeywordEngine.JinaConfig.APIKey
		}
		if (res.BaseURL == "" || res.BaseURL == "https://r.jina.ai/") && a.KeywordEngine.JinaConfig.BaseURL != "" {
			res.BaseURL = a.KeywordEngine.JinaConfig.BaseURL
		}
		if res.APIKey != "" {
			a.KeywordEngine.JinaConfig.APIKey = res.APIKey
		}
	}
	if res.BaseURL == "" {
		res.BaseURL = "https://r.jina.ai/"
	}
	return res
}

// CaptureURL fetches web content using Jina Reader, parses it as a chat_log or note card,
// writes it to the Scattering directory, emits EventThoughtCaptured, and returns the *Card.
func (a *App) CaptureURL(urlStr string) (*Card, error) {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return nil, errors.New("empty url")
	}

	ctx := context.Background()
	if a != nil && a.ctx != nil {
		ctx = a.ctx
	}

	jinaCfg := a.getJinaConfig()
	pipeline := capturePipelineFactory(jinaCfg)

	// Fetch & parse content (fast-fails on protected/blocked/empty pages)
	res, err := pipeline.Capture(ctx, urlStr)
	if err != nil {
		// Fast-fail: no dirty files written to disk!
		return nil, err
	}

	home := ""
	if a != nil {
		home = a.Home
	}
	if home == "" {
		_, h, err := config.LoadConfig()
		if err == nil {
			home = h
			if a != nil {
				a.Home = h
			}
		}
	}
	if home == "" {
		return nil, errors.New("home directory not configured")
	}

	targetDir := filepath.Join(home, "Scattering")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	slugSrc := res.Title
	if slugSrc == "" {
		slugSrc = "web"
	}
	slug := sanitizeSlug(slugSrc)
	if slug == "" {
		slug = res.ID
	}

	baseName := fmt.Sprintf("%s-%s", res.CreatedAt.Format("20060102-1504"), slug)
	cardTypeStr := string(res.Type)

	var targetPath string
	for i := 1; i <= 100; i++ {
		suffix := ""
		if i > 1 {
			suffix = fmt.Sprintf("-%d", i)
		}
		candidate := filepath.Join(targetDir, fmt.Sprintf("%s%s.%s.md", baseName, suffix, cardTypeStr))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			targetPath = candidate
			break
		}
	}

	if targetPath == "" {
		return nil, errors.New("could not find a unique filename")
	}

	// Write file using watcher
	err = watcher.GetWatcher().WriteFile(targetPath, []byte(res.FullMarkdown), cardTypeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to write captured card file: %w", err)
	}

	// Publish EventThoughtCaptured
	bus.GetBus().Publish(bus.Event{
		Type: bus.EventThoughtCaptured,
		Data: bus.EventThoughtCapturedData{
			DocID:        res.ID,
			FilePath:     targetPath,
			Text:         res.Content,
			CapturedAt:   res.CreatedAt,
			Title:        res.Title,
			Type:         cardTypeStr,
			Participants: res.Participants,
			StartedAt:    res.StartedAt,
			EndedAt:      res.EndedAt,
		},
	})

	card := &Card{
		ID:           res.ID,
		Type:         CardType(res.Type),
		Title:        res.Title,
		Content:      res.Content,
		SourceURL:    res.SourceURL,
		CreatedAt:    res.CreatedAt,
		FilePath:     targetPath,
		Participants: res.Participants,
		StartedAt:    res.StartedAt,
		EndedAt:      res.EndedAt,
	}

	return card, nil
}
