package curio

import (
	"context"
	"fmt"
	"Finnegan/core/module"
)

type CurioModule struct {}

var _ module.CommandProvider = (*CurioModule)(nil)
var _ module.Module = (*CurioModule)(nil)
var _ module.ManifestProvider = (*CurioModule)(nil)

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
	return []module.CommandDef{}
}

func (m *CurioModule) ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("unknown command: %s", cmdID)
}

func (m *CurioModule) GetManifest() module.ModuleManifest {
	return module.ModuleManifest{
		Name: "curio",
		CardTypes: []module.CardTypeSpec{
			{Type: "curio"},
		},
		Commands: m.GetCommands(),
	}
}

func init() {
	module.Register(&CurioModule{})
}
