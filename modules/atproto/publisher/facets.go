package publisher

import (
	"regexp"
)

// Facet represents an AT Proto Rich Text facet.
type Facet struct {
	Index    FacetIndex   `json:"index"`
	Features []FacetFeature `json:"features"`
}

type FacetIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type FacetFeature struct {
	Type string `json:"$type"`
	URI  string `json:"uri,omitempty"`
	DID  string `json:"did,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

var (
	urlRegex     = regexp.MustCompile(`https?://[^\s]+`)
	mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9.-]+)`)
	tagRegex     = regexp.MustCompile(`#[a-zA-Z0-9_]+`)
)

func ParseFacets(text string) []Facet {
	var facets []Facet
	textBytes := []byte(text)

	// Mentions
	// In a real app we'd need to resolve handles to DIDs for mentions, but for now we'll mock or just construct the feature
	// Note: We need a DID for a mention in ATProto, but since we don't have a resolver, we'll store the handle in DID for now
	// or assume we use some resolution elsewhere.

	// Actually, wait, let's just create features.
	for _, match := range mentionRegex.FindAllSubmatchIndex(textBytes, -1) {
		start, end := match[0], match[1]
		handle := string(textBytes[match[2]:match[3]])

		// For proper API compatibility without a DID resolver, prefixing with did:mock: is safer, or assuming did:plc:...
		// We'll mock a DID if it's just a handle for tests/simulations to prevent Bluesky from rejecting it instantly.
		facets = append(facets, Facet{
			Index: FacetIndex{ByteStart: start, ByteEnd: end},
			Features: []FacetFeature{
				{
					Type: "app.bsky.richtext.facet#mention",
					DID:  "did:mock:" + handle, // Ideally resolved to real DID, mock for now
				},
			},
		})
	}

	for _, match := range urlRegex.FindAllIndex(textBytes, -1) {
		start, end := match[0], match[1]
		url := string(textBytes[start:end])
		facets = append(facets, Facet{
			Index: FacetIndex{ByteStart: start, ByteEnd: end},
			Features: []FacetFeature{
				{
					Type: "app.bsky.richtext.facet#link",
					URI:  url,
				},
			},
		})
	}

	for _, match := range tagRegex.FindAllIndex(textBytes, -1) {
		start, end := match[0], match[1]
		// Skip the '#'
		tag := string(textBytes[start+1 : end])
		facets = append(facets, Facet{
			Index: FacetIndex{ByteStart: start, ByteEnd: end},
			Features: []FacetFeature{
				{
					Type: "app.bsky.richtext.facet#tag",
					Tag:  tag,
				},
			},
		})
	}

	return facets
}
