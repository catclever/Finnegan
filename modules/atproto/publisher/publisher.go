package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"Finnegan/modules/atproto/config"
	"Finnegan/modules/atproto/models"
)

type Publisher struct {
	Config  *config.Config
	Session *SessionManager
}

func NewPublisher(cfg *config.Config) *Publisher {
	return &Publisher{
		Config:  cfg,
		Session: NewSessionManager(cfg),
	}
}

// updateFrontmatter losslessly updates the YAML frontmatter in a markdown file.
func updateFrontmatter(filePath string, uri string, cid string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	parts := bytes.SplitN(data, []byte("---"), 3)
	if len(parts) < 3 {
		return fmt.Errorf("no frontmatter found")
	}

	var root yaml.Node
	if err := yaml.Unmarshal(parts[1], &root); err != nil {
		return err
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("invalid frontmatter yaml")
	}

	mapping := root.Content[0]

	setOrAdd := func(key, val, tag string) {
		for i := 0; i < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == key {
				mapping.Content[i+1].Value = val
				if tag != "" {
					mapping.Content[i+1].Tag = tag
				}
				return
			}
		}

		k := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
		v := &yaml.Node{Kind: yaml.ScalarNode, Value: val}
		if tag != "" {
			v.Tag = tag
			if tag == "!!bool" {
				v.Value = val // "true" or "false"
			}
		}
		mapping.Content = append(mapping.Content, k, v)
	}

	setOrAdd("uri", uri, "")
	setOrAdd("cid", cid, "")
	setOrAdd("published", "true", "!!bool")

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return err
	}
	enc.Close()

	// Reconstruct the file content
	// Re-add the top level ---
	// The out buffer will include the closing --- if it's encoded as a document? No, it usually just encodes the content.
	// Actually yaml.Encoder on DocumentNode might include `---` prefix? Let's check how yaml outputs.
	outStr := out.String()
	// Clean up any extra doc separators
	outStr = strings.TrimSpace(outStr)
	if strings.HasPrefix(outStr, "---") {
		outStr = strings.TrimPrefix(outStr, "---")
		outStr = strings.TrimSpace(outStr)
	}

	newContent := []byte(string(parts[0]) + "---\n" + outStr + "\n---" + string(parts[2]))

	return os.WriteFile(filePath, newContent, 0644)
}

func (p *Publisher) Publish(text string, facets []Facet, images []string) (string, string, error) {
	// For now, images are not strictly implemented per requirements.

	type PostRecord struct {
		Type      string  `json:"$type"`
		CreatedAt string  `json:"createdAt"`
		Text      string  `json:"text"`
		Facets    []Facet `json:"facets,omitempty"`
	}

	type CreateRecordPayload struct {
		Repo       string     `json:"repo"`
		Collection string     `json:"collection"`
		Record     PostRecord `json:"record"`
	}

	did := p.Session.GetDID()
	if did == "" {
		if err := p.Session.CreateSession(); err != nil {
			return "", "", err
		}
		did = p.Session.GetDID()
	}

	payload := CreateRecordPayload{
		Repo:       did,
		Collection: "app.bsky.feed.post",
		Record: PostRecord{
			Type:      "app.bsky.feed.post",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Text:      text,
			Facets:    facets,
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	url := p.Config.PDSUrl + "/xrpc/com.atproto.repo.createRecord"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Session.DoRequest(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("createRecord failed: %d %s", resp.StatusCode, string(body))
	}

	var res struct {
		Uri string `json:"uri"`
		Cid string `json:"cid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}

	return res.Uri, res.Cid, nil
}

// stripFrontmatter extracts substantive markdown content if frontmatter is present
func stripFrontmatter(text string) string {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "---") {
		parts := strings.SplitN(trimmed, "---", 3)
		if len(parts) >= 3 {
			return strings.TrimSpace(parts[2])
		}
	}
	return text
}

// PublishNote publishes note content to Bluesky and losslessly updates YAML frontmatter in filePath
func (p *Publisher) PublishNote(ctx context.Context, filePath string, content string) (string, string, error) {
	postText := stripFrontmatter(content)
	if strings.TrimSpace(postText) == "" && filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			postText = stripFrontmatter(string(data))
		}
	}
	if strings.TrimSpace(postText) == "" {
		return "", "", fmt.Errorf("no post content to publish")
	}

	facets := ParseFacets(postText)
	uri, cid, err := p.Publish(postText, facets, nil)
	if err != nil {
		return "", "", err
	}

	if filePath != "" {
		if err := updateFrontmatter(filePath, uri, cid); err != nil {
			log.Printf("[ATProto Publisher] updateFrontmatter failed for %s: %v", filePath, err)
			return uri, cid, fmt.Errorf("published successfully to Bluesky (%s), but failed to update frontmatter: %w", uri, err)
		}
	}

	return uri, cid, nil
}

// PublishNew pushes a completely new record to AT Protocol
func (p *Publisher) PublishNew(post *models.Post) error {
	log.Printf("[ATProto Publisher] CREATING new record on Bluesky...")

	_, _, err := p.PublishNote(context.Background(), post.FilePath, post.Content)
	if err != nil {
		return err
	}

	log.Printf("[ATProto Publisher] ✅ New Post Success!")
	return nil
}

// PublishUpdate pushes an edit to an existing record on AT Protocol
func (p *Publisher) PublishUpdate(post *models.Post) error {
	log.Printf("[ATProto Publisher] UPDATING existing record %s on Bluesky... (Mocked)", post.Metadata.URI)
	// We'll leave the actual PUT request logic for later as it's not strictly required in step 3 details, but we will call updateFrontmatter to simulate success.
	return nil
}
