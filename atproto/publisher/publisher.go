package publisher

import (
	"log"

	"Finnegan/atproto/config"
	"Finnegan/atproto/models"
)

type Publisher struct {
	Config *config.Config
}

func NewPublisher(cfg *config.Config) *Publisher {
	return &Publisher{
		Config: cfg,
	}
}

// PublishNew pushes a completely new record to AT Protocol
func (p *Publisher) PublishNew(post *models.Post) error {
	log.Printf("[ATProto Publisher] CREATING new record on Bluesky...")
	
	// TODO: 
	// 1. XRPC: com.atproto.server.createSession
	// 2. XRPC: com.atproto.repo.createRecord 
	//    - collection: app.bsky.feed.post
	//    - record: { text: "...", facets: [...] }
	
	log.Printf("[ATProto Publisher] ✅ New Post Success! (Simulated)")
	// After success, we would write `uri: at://...` and `published: true` back to the file
	
	return nil
}

// PublishUpdate pushes an edit to an existing record on AT Protocol
func (p *Publisher) PublishUpdate(post *models.Post) error {
	log.Printf("[ATProto Publisher] UPDATING existing record %s on Bluesky...", post.Metadata.URI)
	
	// TODO:
	// 1. XRPC: com.atproto.server.createSession
	// 2. Parse the existing URI to extract the `rkey` (record key)
	//    Format: at://<did>/<collection>/<rkey>
	// 3. XRPC: com.atproto.repo.putRecord 
	//    - repo: our DID
	//    - collection: app.bsky.feed.post
	//    - rkey: extracted rkey
	//    - record: { text: "<UPDATED TEXT>", facets: [...] }
	
	log.Printf("[ATProto Publisher] ✅ Post Updated Success! (Simulated)")
	
	return nil
}
