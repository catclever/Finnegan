package module

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// LLMConfig represents the network-level LLM configuration from the global config
type LLMConfig struct {
	Provider     string `yaml:"provider" json:"provider"`
	APIKey       string `yaml:"api_key" json:"api_key"`
	BaseURL      string `yaml:"base_url" json:"base_url"`
	DefaultModel string `yaml:"default_model" json:"default_model"`
}

func (c *LLMConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]string
	if err := value.Decode(&raw); err != nil {
		return err
	}
	for k, v := range raw {
		normKey := strings.ToLower(strings.ReplaceAll(k, "_", ""))
		switch normKey {
		case "provider":
			c.Provider = v
		case "apikey", "key":
			c.APIKey = v
		case "baseurl", "url":
			c.BaseURL = v
		case "defaultmodel", "model":
			c.DefaultModel = v
		}
	}
	return nil
}

// StorageInterface represents a mock storage interface for modules
type StorageInterface interface{}

// WatcherInterface represents a mock file watcher bus
type WatcherInterface interface{}

// Context is the environment context passed from Core to Modules
type Context struct {
	LLMs    map[string]LLMConfig
	Storage StorageInterface
	Watcher WatcherInterface
}

// Module represents the standard interface that all Finnegan modules must implement
type Module interface {
	Name() string
	ConfigTarget() interface{}
	Init(ctx *Context) error
	Bindings() []interface{}
}

var registry = make(map[string]Module)

// Register adds a module to the global registry
func Register(m Module) {
	registry[m.Name()] = m
}

// GetRegistry returns all registered modules
func GetRegistry() map[string]Module {
	return registry
}
