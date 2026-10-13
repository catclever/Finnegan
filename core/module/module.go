package module

// LLMConfig represents the network-level LLM configuration from the global config
type LLMConfig struct {
	Provider     string
	APIKey       string
	BaseURL      string
	DefaultModel string
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
