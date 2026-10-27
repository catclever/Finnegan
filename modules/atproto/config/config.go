package config

// Config represents the connection and environment settings for AT Protocol.
// Values will be injected by Finnegan Core (4-Tier configuration).
type Config struct {
	Handle    string `yaml:"handle"`
	DID       string `yaml:"did"`
	Password  string `yaml:"password"`
	PDSUrl    string `yaml:"pds_url"`
	InboxDir  string `yaml:"inbox_dir"`
	NotesDir  string `yaml:"notes_dir"`
}

func (c *Config) GetPDSUrl() string {
	if c.PDSUrl == "" {
		return "https://bsky.social"
	}
	return c.PDSUrl
}
