package publisher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"Finnegan/modules/atproto/config"
	"Finnegan/modules/atproto/models"
)

type Publisher struct {
	Config *config.Config
	Auth   *AuthManager
}

func NewPublisher(cfg *config.Config) *Publisher {
	return &Publisher{
		Config: cfg,
		Auth:   NewAuthManager(cfg),
	}
}

type Record struct {
	Type      string   `json:"$type"`
	Text      string   `json:"text"`
	CreatedAt string   `json:"createdAt"`
	Facets    []Facet  `json:"facets,omitempty"`
}

type CreateRecordRequest struct {
	Repo       string `json:"repo"`
	Collection string `json:"collection"`
	Record     Record `json:"record"`
}

type PutRecordRequest struct {
	Repo       string `json:"repo"`
	Collection string `json:"collection"`
	RKey       string `json:"rkey"`
	Record     Record `json:"record"`
}

type CreateRecordResponse struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

func (p *Publisher) Publish(text string, facets []Facet, images []string) (uri string, cid string, err error) {
	// For simplicity, we are ignoring images in this step per the constraints.

	url := fmt.Sprintf("%s/xrpc/com.atproto.repo.createRecord", p.Config.GetPDSUrl())

	// Identifier can be either handle or DID. AuthManager should resolve DID on login,
	// but we must use DID for 'repo'. Let's default to DID from config or session.
	did := p.Config.DID
	if did == "" && p.Auth.Session != nil {
		did = p.Auth.Session.DID
	}
	// If still empty, assume identifier was a DID and use Handle
	if did == "" {
		did = p.Config.Handle
	}

	payload := CreateRecordRequest{
		Repo:       did,
		Collection: "app.bsky.feed.post",
		Record: Record{
			Type:      "app.bsky.feed.post",
			Text:      text,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Facets:    facets,
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Auth.DoAuthenticatedRequest(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", "", fmt.Errorf("failed to publish record: status %d, err: %v", resp.StatusCode, errResp)
	}

	var res CreateRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}

	return res.URI, res.CID, nil
}

// PublishNew pushes a completely new record to AT Protocol
func (p *Publisher) PublishNew(post *models.Post) error {
	log.Printf("[ATProto Publisher] CREATING new record on Bluesky...")

	facets := ParseFacets(post.Content)
	uri, cid, err := p.Publish(post.Content, facets, nil)
	if err != nil {
		return err
	}
	
	log.Printf("[ATProto Publisher] ✅ New Post Success! URI: %s", uri)

	// Update the file frontmatter losslessly
	if err := UpdateFrontmatter(post.FilePath, uri, cid); err != nil {
		return fmt.Errorf("published successfully but failed to update frontmatter: %w", err)
	}
	
	// Update in-memory model too
	post.Metadata.URI = uri
	post.Metadata.CID = cid
	post.Metadata.Published = true

	return nil
}

// PublishUpdate pushes an edit to an existing record on AT Protocol
func (p *Publisher) PublishUpdate(post *models.Post) error {
	log.Printf("[ATProto Publisher] UPDATING existing record %s on Bluesky...", post.Metadata.URI)
	
	url := fmt.Sprintf("%s/xrpc/com.atproto.repo.putRecord", p.Config.GetPDSUrl())

	did := p.Config.DID
	if did == "" && p.Auth.Session != nil {
		did = p.Auth.Session.DID
	}
	if did == "" {
		did = p.Config.Handle
	}

	// Format: at://<did>/<collection>/<rkey>
	parts := strings.Split(post.Metadata.URI, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid URI format: %s", post.Metadata.URI)
	}
	rkey := parts[len(parts)-1]

	payload := PutRecordRequest{
		Repo:       did,
		Collection: "app.bsky.feed.post",
		RKey:       rkey,
		Record: Record{
			Type:      "app.bsky.feed.post",
			Text:      post.Content,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Facets:    ParseFacets(post.Content),
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Auth.DoAuthenticatedRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to update record: status %d, err: %v", resp.StatusCode, errResp)
	}

	var res CreateRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		post.Metadata.CID = res.CID // cid usually updates on putRecord
		_ = UpdateFrontmatter(post.FilePath, res.URI, res.CID)
	}

	log.Printf("[ATProto Publisher] ✅ Post Updated Success!")
	
	return nil
}
