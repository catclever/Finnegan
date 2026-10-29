package babel

import (
	"context"
	"fmt"
	"strings"

	coremodels "Finnegan/core/models"
	"Finnegan/core/module"
)

// Pipeline orchestrates Universal Base filters, language translation, and layered gatekeepers.
type Pipeline struct {
	Translator  *UniversalTranslator
	gatekeepers map[string]Gatekeeper
}

// NewPipeline creates a new Babel pipeline.
func NewPipeline(llmCfg module.LLMConfig, modelName string, fn CompletionFunc) *Pipeline {
	translator := NewUniversalTranslator(llmCfg, modelName, fn)
	p := &Pipeline{
		Translator:  translator,
		gatekeepers: make(map[string]Gatekeeper),
	}

	// Register default gatekeepers
	p.RegisterGatekeeper(NewBuguGatekeeper(translator))
	p.RegisterGatekeeper(NewBlueskyGatekeeper(translator))

	return p
}

// RegisterGatekeeper registers a domain gatekeeper by its origin identifier (e.g. "bugu", "atproto").
func (p *Pipeline) RegisterGatekeeper(g Gatekeeper) {
	if g != nil {
		p.gatekeepers[g.Origin()] = g
	}
}

// Process routes input through the origin-appropriate gatekeeper with universal base filtering and translation.
func (p *Pipeline) Process(ctx context.Context, input RawBabelInput) (*coremodels.BabelCard, error) {
	origin := strings.TrimSpace(input.Origin)
	if origin == "" {
		origin = "bugu"
	}

	gk, ok := p.gatekeepers[origin]
	if !ok {
		// Fallback to bugu gatekeeper or default curation
		if origin == "atproto" || origin == "bluesky" {
			gk = p.gatekeepers["atproto"]
		} else {
			gk = p.gatekeepers["bugu"]
		}
	}

	if gk == nil {
		return nil, fmt.Errorf("no gatekeeper registered for origin: %s", origin)
	}

	return gk.ValidateAndCurate(ctx, input)
}
