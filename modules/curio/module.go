package curio

import (
	"context"
	"fmt"
	"Finnegan/core/module"
)

type CurioModule struct {}

var _ module.CommandProvider = (*CurioModule)(nil)
var _ module.Module = (*CurioModule)(nil)

func (m *CurioModule) Name() string {
	return "curio"
}

func (m *CurioModule) ConfigTarget() interface{} {
	return nil
}

func (m *CurioModule) Init(ctx *module.Context) error {
	return nil
}

func (m *CurioModule) Bindings() []interface{} {
	return []interface{}{
		m,
	}
}

func (m *CurioModule) GetCommands() []module.CommandDef {
	return []module.CommandDef{
		{
			ID:          "curio:ingest",
			Name:        "Curio Ingestion",
			Trigger:     "curio",
			Description: "Ingest a screenshot for Curio",
			Namespace:   "curio",
			IsPrimary:   true,
			Params: []module.ParamSpec{
				{
					Name:        "screenshot",
					Type:        module.ParamTypeText,
					Description: "Screenshot path or payload",
				},
			},
		},
	}
}

func (m *CurioModule) ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error) {
	if cmdID == "curio:ingest" {
		return map[string]interface{}{
			"status":  "triggered",
			"command": cmdID,
		}, nil
	}
	return nil, fmt.Errorf("unknown command: %s", cmdID)
}

func init() {
	module.Register(&CurioModule{})
}
