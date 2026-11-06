package diary

import (
	"context"
	"Finnegan/core/module"
)

type DiaryModule struct{}

func init() {
	module.Register(&DiaryModule{})
}

func (m *DiaryModule) Name() string {
	return "diary"
}

func (m *DiaryModule) ConfigTarget() interface{} {
	return nil
}

func (m *DiaryModule) Init(ctx *module.Context) error {
	return nil
}

func (m *DiaryModule) Bindings() []interface{} {
	return nil
}

func (m *DiaryModule) GetManifest() module.ModuleManifest {
	return module.ModuleManifest{
		Name:       "diary",
		ThemeColor: "#d97706",
		CardTypes: []module.CardTypeSpec{
			{Type: "diary"},
		},
	}
}

func (m *DiaryModule) GetCommands() []module.CommandDef {
	return []module.CommandDef{
		{
			ID:          "diary:search",
			Name:        "羊皮卷日记检索",
			Trigger:     "diary",
			Namespace:   "diary",
			IsPrimary:   true,
			Description: "按精确小时或时间范围检索羊皮卷日记",
			Params: []module.ParamSpec{
				{
					Name: "query",
					Type: module.ParamTypeText,
				},
			},
		},
	}
}

func (m *DiaryModule) ExecuteCommand(ctx context.Context, cmdID string, params map[string]interface{}) (interface{}, error) {
	return nil, nil
}
