package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadLocalCards scans the home directory's Scattering and Weaving folders and returns parsed local cards
func (a *App) LoadLocalCards() []Card {
	var localCards []Card

	dirs := []string{
		filepath.Join(a.Home, "Scattering"),
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
			
			// Remove frontmatter if present
			if strings.HasPrefix(content, "---") {
				parts := strings.SplitN(content, "---", 3)
				if len(parts) >= 3 {
					card.Content = strings.TrimSpace(parts[2])
				}
			}

			localCards = append(localCards, card)
		}
	}

	return localCards
}
