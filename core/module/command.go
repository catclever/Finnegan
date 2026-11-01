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
	Name                string    `json:"name" yaml:"name"`
	Type                ParamType `json:"type" yaml:"type"`
	Description         string    `json:"description" yaml:"description"`
	DataSource          string    `json:"dataSource" yaml:"data_source,omitempty"`
	RequireKeywords     bool      `json:"requireKeywords,omitempty" yaml:"require_keywords,omitempty"`
	ExcludedTypes       []string  `json:"excludedTypes,omitempty" yaml:"excluded_types,omitempty"`
	AllowedTypes        []string  `json:"allowedTypes,omitempty" yaml:"allowed_types,omitempty"`
	AutoStageToViewport bool      `json:"autoStageToViewport,omitempty" yaml:"auto_stage_to_viewport,omitempty"`
}

// CommandDef defines an Omnibar command provided by a module
type CommandDef struct {
	ID          string      `json:"id" yaml:"id"`
	Name        string      `json:"name" yaml:"name"`
	Trigger     string      `json:"trigger" yaml:"trigger"`
	Description string      `json:"description" yaml:"description"`
	Params      []ParamSpec `json:"params" yaml:"params,omitempty"`
	Namespace   string      `json:"namespace,omitempty" yaml:"namespace,omitempty"`   // 模块专名 (e.g. "bugu", "bsky")
	IsPrimary   bool        `json:"isPrimary,omitempty" yaml:"is_primary,omitempty"`  // 是否为专名主命令
	SubCommand  string      `json:"subCommand,omitempty" yaml:"sub_command,omitempty"` // 子命令标识 (e.g. "push")
}

// CommandProvider represents the interface modules implement to expose Omnibar commands
type CommandProvider interface {
	GetCommands() []CommandDef
	ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error)
}
