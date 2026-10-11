package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the connection and environment settings for AT Protocol
type Config struct {
	DID       string `yaml:"did"`
	Password  string `yaml:"password"`
	PDSUrl    string `yaml:"pds_url"`
	InboxDir  string `yaml:"inbox_dir"`
	NotesDir  string `yaml:"notes_dir"`
}

// LoadConfig reads and parses the AT Protocol configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
