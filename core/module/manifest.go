package module

import "context"

// PromotionStrategySpec defines how temporary cards spawned by a module are promoted to permanent vault cards.
type PromotionStrategySpec struct {
	Icon             string `json:"icon" yaml:"icon"`                         // e.g. "default_book", "anchor_comment"
	Tooltip          string `json:"tooltip" yaml:"tooltip"`                   // e.g. "附着母卡收录（附带批注）"
	HasCommentPrompt bool   `json:"hasCommentPrompt" yaml:"has_comment_prompt"`
	Placeholder      string `json:"placeholder" yaml:"placeholder"`
	BindMotherCard   bool   `json:"bindMotherCard" yaml:"bind_mother_card"`
}

// CardTypeSpec declares a custom card type and optional frontend UI bundle entry for a module.
type CardTypeSpec struct {
	Type    string `json:"type" yaml:"type"`                           // e.g. "curio", "taohuo"
	UIEntry string `json:"uiEntry,omitempty" yaml:"ui_entry,omitempty"` // e.g. "/api/plugins/taohuo/ui.js"
}

// ModuleManifest declares the full metadata, UI theme, beacon, promotion strategy, card types, and commands for a module.
type ModuleManifest struct {
	Name              string                 `json:"name" yaml:"name"`
	Version           string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Description       string                 `json:"description,omitempty" yaml:"description,omitempty"`
	ThemeColor        string                 `json:"themeColor,omitempty" yaml:"theme_color,omitempty"` // e.g. "#f59e0b" (amber), "#38bdf8" (sky)
	BeaconType        string                 `json:"beaconType,omitempty" yaml:"beacon_type,omitempty"` // e.g. "radar", "butterfly", "none"
	PromotionStrategy *PromotionStrategySpec `json:"promotionStrategy,omitempty" yaml:"promotion_strategy,omitempty"`
	CardTypes         []CardTypeSpec         `json:"cardTypes,omitempty" yaml:"card_types,omitempty"`
	Commands          []CommandDef           `json:"commands,omitempty" yaml:"commands,omitempty"`
	IsExternal        bool                   `json:"isExternal,omitempty" yaml:"-"`
	Entry             string                 `json:"-" yaml:"entry,omitempty"`
}

// ManifestProvider can be implemented by any Module to self-declare its ModuleManifest.
type ManifestProvider interface {
	GetManifest() ModuleManifest
}

// ActionProvider can be implemented by any Module to handle custom RPC actions from the frontend or event bus.
type ActionProvider interface {
	ExecuteAction(ctx context.Context, actionID string, payload map[string]interface{}) (interface{}, error)
}

// PromoteHookProvider can be implemented by a Module to customize card promotion data.
type PromoteHookProvider interface {
	OnCardPromoted(ctx context.Context, card map[string]interface{}, comment string, echoOf string) (map[string]interface{}, error)
}

// BuildModuleManifest inspects a Module and constructs a normalized ModuleManifest,
// applying backward-compatible defaults for built-in modules that have not yet implemented ManifestProvider.
func BuildModuleManifest(m Module) ModuleManifest {
	if m == nil {
		return ModuleManifest{}
	}

	var manifest ModuleManifest
	if mp, ok := m.(ManifestProvider); ok {
		manifest = mp.GetManifest()
	}

	if manifest.Name == "" {
		manifest.Name = m.Name()
	}

	if cp, ok := m.(CommandProvider); ok && len(manifest.Commands) == 0 {
		manifest.Commands = cp.GetCommands()
	}

	// Backward-compatible defaults for built-in modules
	switch manifest.Name {
	case "bugu":
		if manifest.ThemeColor == "" {
			manifest.ThemeColor = "#f59e0b"
		}
		if manifest.BeaconType == "" {
			manifest.BeaconType = "radar"
		}
		if manifest.PromotionStrategy == nil {
			manifest.PromotionStrategy = &PromotionStrategySpec{
				Icon:             "anchor_comment",
				Tooltip:          "附着母卡收录（附带批注）",
				HasCommentPrompt: true,
				Placeholder:      "留下附着批注（回车直接收录）...",
				BindMotherCard:   true,
			}
		}
	case "atproto":
		if manifest.ThemeColor == "" {
			manifest.ThemeColor = "#38bdf8"
		}
		if manifest.BeaconType == "" {
			manifest.BeaconType = "butterfly"
		}
		if manifest.PromotionStrategy == nil {
			manifest.PromotionStrategy = &PromotionStrategySpec{
				Icon:             "anchor_comment",
				Tooltip:          "附着母卡收录（附带批注）",
				HasCommentPrompt: true,
				Placeholder:      "留下附着批注（回车直接收录）...",
				BindMotherCard:   true,
			}
		}
	case "curio":
		if len(manifest.CardTypes) == 0 {
			manifest.CardTypes = []CardTypeSpec{{Type: "curio"}}
		}
	}

	return manifest
}
