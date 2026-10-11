package markdown

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ParsedDoc struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	CreatedAt time.Time      `json:"createdAt"`
	Meta      map[string]any `json:"meta"`
	Body      string         `json:"body"`
}

func ParseFile(path string, existingCreatedAt time.Time) (*ParsedDoc, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, content, existingCreatedAt)
}

func FallbackID(absPath string) string {
	hash := sha1.Sum([]byte(absPath))
	return hex.EncodeToString(hash[:])
}

func Parse(path string, content []byte, existingCreatedAt time.Time) (*ParsedDoc, error) {
	strContent := string(content)
	
	var meta map[string]any
	var body string
	
	if strings.HasPrefix(strContent, "---\n") || strings.HasPrefix(strContent, "---\r\n") {
		parts := strings.SplitN(strContent, "---", 3)
		if len(parts) >= 3 {
			yamlStr := parts[1]
			body = strings.TrimPrefix(parts[2], "\n")
			body = strings.TrimPrefix(body, "\r\n")
			
			if err := yaml.Unmarshal([]byte(yamlStr), &meta); err != nil {
				meta = make(map[string]any)
				body = strContent
			}
		} else {
			meta = make(map[string]any)
			body = strContent
		}
	} else {
		meta = make(map[string]any)
		body = strContent
	}
	
	if meta == nil {
		meta = make(map[string]any)
	}

	doc := &ParsedDoc{
		Meta: meta,
		Body: body,
	}

	if id, ok := meta["id"].(string); ok && id != "" {
		doc.ID = id
	} else {
		abs, _ := filepath.Abs(path)
		doc.ID = FallbackID(abs)
	}
	
	if title, ok := meta["title"].(string); ok && title != "" {
		doc.Title = title
	} else {
		lines := strings.Split(body, "\n")
		found := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				doc.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
				found = true
				break
			}
		}
		if !found {
			base := filepath.Base(path)
			doc.Title = strings.TrimSuffix(base, filepath.Ext(base))
		}
	}
	
	if createdTime, ok := meta["created"].(time.Time); ok {
		doc.CreatedAt = createdTime
	} else if createdStr, ok := meta["created"].(string); ok && createdStr != "" {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			doc.CreatedAt = t
		}
	}
	
	if doc.CreatedAt.IsZero() {
		if !existingCreatedAt.IsZero() {
			doc.CreatedAt = existingCreatedAt
		} else {
			if stat, err := os.Stat(path); err == nil {
				doc.CreatedAt = stat.ModTime()
			} else {
				doc.CreatedAt = time.Now()
			}
		}
	}
	
	return doc, nil
}
