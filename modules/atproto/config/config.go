package config

// Config represents the connection and environment settings for AT Protocol.
// Values will be injected by Finnegan Core (4-Tier configuration).
type Config struct {
	Handle      string `yaml:"handle"`       // Or DID
	DID         string `yaml:"did"`
	AppPassword string `yaml:"app_password"` // Or password
	Password    string `yaml:"password"`
	PDSUrl      string `yaml:"pds_url"`      // Default https://bsky.social
	InboxDir    string `yaml:"inbox_dir"`
	NotesDir    string `yaml:"notes_dir"`
}
