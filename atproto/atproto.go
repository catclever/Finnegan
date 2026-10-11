package atproto

import (
	"log"

	"Finnegan/atproto/config"
	"Finnegan/atproto/publisher"
)

type Engine struct {
	Config    *config.Config
	Publisher *publisher.Publisher
}

// Start initializes the AT Protocol Bridge
func Start(cfgPath string) *Engine {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Printf("⚠️  [ATProto] Config missing at %s, remaining dormant.", cfgPath)
		return nil
	}

	pub := publisher.NewPublisher(cfg)

	engine := &Engine{
		Config:    cfg,
		Publisher: pub,
	}

	log.Printf("✅ AT Protocol Bridge ready.")
	return engine
}
