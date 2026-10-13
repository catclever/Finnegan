package module

import "context"

// ParamType defines the supported parameter types for module commands
type ParamType string

const (
	ParamTypeNone ParamType = "none"
	ParamTypeCard ParamType = "card"
	ParamTypeText ParamType = "text"
)

// ParamSpec specifies an expected parameter for a module command
type ParamSpec struct {
	Name        string    `json:"name"`
	Type        ParamType `json:"type"`
	Description string    `json:"description"`
	DataSource  string    `json:"dataSource"`
}

// CommandDef defines an Omnibar command provided by a module
type CommandDef struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Trigger     string      `json:"trigger"`
	Description string      `json:"description"`
	Params      []ParamSpec `json:"params"`
}

// CommandProvider represents the interface modules implement to expose Omnibar commands
type CommandProvider interface {
	GetCommands() []CommandDef
	ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error)
}
