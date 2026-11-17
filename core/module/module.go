package module

import (
	"strings"
	"sync"

	"Finnegan/core/models"
	"gopkg.in/yaml.v3"
)

// HostAPI provides callbacks for modules (built-in or external) to interact with the Finnegan host runtime.
type HostAPI interface {
	SpawnCard(card models.BabelCard)
	CompleteSession(cardID string)
	EmitEvent(eventName string, data interface{})
}

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
	HomeDir string
	LLMs    map[string]LLMConfig
	Storage StorageInterface
	Watcher WatcherInterface
	Host    HostAPI
}

// Module represents the standard interface that all Finnegan modules must implement
type Module interface {
	Name() string
	ConfigTarget() interface{}
	Init(ctx *Context) error
	Bindings() []interface{}
}

var (
	registryMu      sync.RWMutex
	registry        = make(map[string]Module)
	builtinRegistry = make(map[string]Module)
)

// Register adds a module to the global registry.
// Non-external modules are also recorded in builtinRegistry so ResetExternalPlugins can restore them.
func Register(m Module) {
	if m == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[m.Name()] = m
	if _, isExt := m.(*ExternalModule); !isExt {
		builtinRegistry[m.Name()] = m
	}
}

// ResetExternalPlugins resets the active registry to a fresh copy of builtinRegistry
// so removed external plugins are cleanly unregistered on hot-reload while built-in modules are restored.
func ResetExternalPlugins() {
	registryMu.Lock()
	defer registryMu.Unlock()
	fresh := make(map[string]Module, len(builtinRegistry))
	for k, v := range builtinRegistry {
		fresh[k] = v
	}
	registry = fresh
}

// GetRegistry returns a snapshot of all registered modules.
func GetRegistry() map[string]Module {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make(map[string]Module, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}
