package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadLocalCards scans the home directory's Scattering and Weaving folders and returns parsed local cards
func (a *App) LoadLocalCards() []Card {
	var localCards []Card

	dirs := []string{
		filepath.Join(a.Home, "Scattering"),
		filepath.Join(a.Home, "Waving"),
		filepath.Join(a.Home, "Weaving"),
	}

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}

			path := filepath.Join(dir, f.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			content := string(data)
			// Simple naive parsing for now
			card := Card{
				ID:        f.Name(),
				Type:      CardTypeThought,
				Title:     "Thought",
				Content:   content,
				CreatedAt: time.Now(),
			}
			
			// Remove frontmatter if present and parse metadata
			if strings.HasPrefix(content, "---") {
				parts := strings.SplitN(content, "---", 3)
				if len(parts) >= 3 {
					card.Content = strings.TrimSpace(parts[2])

					var fm struct {
						Images []string `yaml:"images"`
					}
					if err := yaml.Unmarshal([]byte(parts[1]), &fm); err == nil && len(fm.Images) > 0 {
						card.Images = fm.Images
					} else {
						var rawMeta map[string]any
						if err := yaml.Unmarshal([]byte(parts[1]), &rawMeta); err == nil {
							if rawImgs, ok := rawMeta["images"].([]any); ok {
								for _, item := range rawImgs {
									if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
										card.Images = append(card.Images, strings.TrimSpace(s))
									}
								}
							}
						}
					}
				}
			}

			// Derive meaningful title from first line of content
			lines := strings.Split(card.Content, "\n")
			for _, l := range lines {
				trimmed := strings.TrimSpace(l)
				if trimmed != "" {
					title := strings.TrimPrefix(trimmed, "# ")
					runes := []rune(title)
					if len(runes) > 24 {
						card.Title = string(runes[:24]) + "..."
					} else {
						card.Title = string(runes)
					}
					break
				}
			}

			localCards = append(localCards, card)
		}
	}

	return localCards
}
