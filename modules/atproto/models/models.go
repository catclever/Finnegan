package models

// PostMetadata represents the YAML frontmatter of a Markdown note in Finnegan
type PostMetadata struct {
	Publish   bool     `yaml:"publish"`             // Trigger to publish
	Published bool     `yaml:"published"`           // Has been published successfully
	URI       string   `yaml:"uri,omitempty"`       // AT Protocol post URI (at://...)
	CID       string   `yaml:"cid,omitempty"`       // AT Protocol post CID
	Tags      []string `yaml:"tags,omitempty"`      // Local tags
}

// Post represents an in-memory card/note waiting to be processed
type Post struct {
	Metadata     PostMetadata
	Content      string // The actual markdown body
	FilePath     string // Absolute path of the source file
	Type         string // e.g., "babel"
	Origin       string // e.g., "atproto"
	SourceURL    string // Link to the original post
	AuthorName   string // Handle of the author
	AuthorAvatar string // Avatar URL of the author
	OriginalText string // The original raw text of the post
	EchoOf       string // The ID of the document this is responding to/echoing
}
