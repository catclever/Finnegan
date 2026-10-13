package atproto

import (
	"log"

	"Finnegan/modules/atproto/config"
	"Finnegan/modules/atproto/fetcher"
	"Finnegan/modules/atproto/publisher"
	"Finnegan/core/module"
)

type AtProtoModule struct {
	cfg       *config.Config
	publisher *publisher.Publisher
	fetcher   *fetcher.Fetcher
	ctx       *module.Context
	API       *AtProtoAPI
}

// AtProtoAPI defines the methods exposed to the Wails frontend (Svelte)
type AtProtoAPI struct {
	publisher *publisher.Publisher
	fetcher   *fetcher.Fetcher
}

func init() {
	module.Register(&AtProtoModule{
		cfg: &config.Config{
			PDSUrl: "https://bsky.social",
		},
	})
}

func (m *AtProtoModule) Name() string {
	return "atproto"
}

func (m *AtProtoModule) ConfigTarget() interface{} {
	return m.cfg
}

func (m *AtProtoModule) Init(ctx *module.Context) error {
	m.ctx = ctx
	m.publisher = publisher.NewPublisher(m.cfg)
	m.fetcher = fetcher.NewFetcher()
	
	m.API = &AtProtoAPI{
		publisher: m.publisher,
		fetcher:   m.fetcher,
	}

	log.Printf("[ATProto] Module ready. Manual trigger mode enabled.")
	return nil
}

// Bindings exposes the API struct to the Wails frontend
func (m *AtProtoModule) Bindings() []interface{} {
	return []interface{}{
		m.API,
	}
}

// ----------------------------------------------------------------------
// Wails Frontend API Methods (These are converted to JS Promises in Svelte)
// ----------------------------------------------------------------------

// PublishNote allows the user to manually trigger publishing a card to Bluesky from the GUI
func (api *AtProtoAPI) PublishNote(filePath string, content string) (string, error) {
	log.Printf("[ATProto API] User manually triggered publish for: %s", filePath)
	// TODO: construct models.Post from content, pass to api.publisher
	return "at://simulated-uri", nil
}

// SearchSimilar allows the user to manually trigger a deep dive search on Bluesky
func (api *AtProtoAPI) SearchSimilar(query string, limit int) (string, error) {
	log.Printf("[ATProto API] User triggered search for: %s", query)
	// TODO: call api.fetcher.SearchSimilar, save to Inbox
	return "Search dispatched", nil
}
