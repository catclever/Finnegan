package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ParseCardFile reads a markdown file from disk, parses YAML frontmatter and body, and returns a Card.
func ParseCardFile(filePath string) (*Card, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	content := string(data)
	fName := filepath.Base(filePath)
	card := Card{
		ID:        fName,
		Type:      CardTypeThought,
		Title:     "Thought",
		Content:   content,
		CreatedAt: time.Now(),
		FilePath:  filePath,
	}

	if fi, err := os.Stat(filePath); err == nil {
		card.CreatedAt = fi.ModTime()
	}

	// Remove frontmatter if present and parse metadata
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			card.Content = strings.TrimSpace(parts[2])

			var fm struct {
				ID           string              `yaml:"id"`
				Type         string              `yaml:"type"`
				Title        string              `yaml:"title"`
				Participants []string            `yaml:"participants"`
				StartedAt    string              `yaml:"started_at"`
				EndedAt      string              `yaml:"ended_at"`
				CreatedAt    string              `yaml:"created_at"`
				Created      string              `yaml:"created"`
				Images       []string            `yaml:"images"`
				Keywords     map[string][]string `yaml:"keywords"`
			}
			if err := yaml.Unmarshal([]byte(parts[1]), &fm); err == nil {
				if fm.ID != "" {
					card.ID = fm.ID
				}
				if fm.Type != "" {
					card.Type = CardType(fm.Type)
				}
				if fm.Title != "" {
					card.Title = fm.Title
				}
				if len(fm.Participants) > 0 {
					card.Participants = fm.Participants
				}
				if fm.StartedAt != "" {
					if t, err := time.Parse(time.RFC3339, fm.StartedAt); err == nil {
						card.StartedAt = &t
					}
				}
				if fm.EndedAt != "" {
					if t, err := time.Parse(time.RFC3339, fm.EndedAt); err == nil {
						card.EndedAt = &t
					}
				}
				if fm.CreatedAt != "" {
					if t, err := time.Parse(time.RFC3339, fm.CreatedAt); err == nil {
						card.CreatedAt = t
					}
				} else if fm.Created != "" {
					if t, err := time.Parse(time.RFC3339, fm.Created); err == nil {
						card.CreatedAt = t
					}
				}
				if len(fm.Images) > 0 {
					card.Images = fm.Images
				}
				if len(fm.Keywords) > 0 {
					card.Keywords = fm.Keywords
				}
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

	// If no explicit title was in frontmatter, derive from first line
	if card.Title == "Thought" || card.Title == "Note" || card.Title == "" {
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
	}

	return &card, nil
}

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
			card, err := ParseCardFile(path)
			if err != nil {
				continue
			}
			localCards = append(localCards, *card)
		}
	}

	sort.SliceStable(localCards, func(i, j int) bool {
		return localCards[i].CreatedAt.After(localCards[j].CreatedAt)
	})

	return localCards
}
