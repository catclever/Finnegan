package keywords

import "strings"

// 4 Concrete Anchor Dimension Constants
const (
	DimensionObject  = "object"  // 名物与通用指涉 (具体人/角色、作品、风物、工具、技术、食物、地点等名物；不仅包含专有名词，也完全包含日常通用指涉如 菜市场, 叔叔, 外卖员, 猫, 设计师, 前端)
	DimensionMotif   = "motif"   // 处境与母题 (人类生活、对话与工作里反复上演的经典剧目、动作切片或行为场景，如 逃席, 吐槽模型, 创造需求, 薅羊毛, 体验新玩具, 额度焦虑)
	DimensionMood    = "mood"    // 心境与心绪 (淬炼成名词的2~4字心境颗粒与审美体验，如 吐槽欲, 通透, 松弛感, 抽离感, 戏谑, 会心一笑, 烦躁, 踏实)
	DimensionConcept = "concept" // 观念与议题 (思想交锋、价值观或判断背后的核心词条/论点抓手，如 伪提效, 创造新需求, 机械迎合, 以Bug改Bug, 前端即逻辑, 平台套路)
)

// Legacy dimension constants retained for backward compatibility
const (
	DimensionEntity           = "entity"
	DimensionEvent            = "event"
	DimensionJudgment         = "judgment"
	DimensionEmotion          = "emotion"
	DimensionQuestion         = "question"
	DimensionMetaphor         = "metaphor"
	DimensionSymptom          = "symptom"
	DimensionMethodology      = "methodology"
	DimensionCounterIntuitive = "counter_intuitive"
	DimensionPersonWork       = "person_work"
)

// DimensionDef defines the specification and prompts for a cognitive dimension.
type DimensionDef struct {
	ID            string `json:"id" yaml:"id"`
	Name          string `json:"name" yaml:"name"`
	Description   string `json:"description" yaml:"description"`
	GatePrompt    string `json:"gate_prompt" yaml:"gate_prompt"`
	ExtractPrompt string `json:"extract_prompt" yaml:"extract_prompt"`
}

// KeywordsProfile represents extracted keywords mapped by dimension ID.
type KeywordsProfile map[string][]string

// Config defines the configuration for the keyword engine.
type Config struct {
	EnabledDimensions []string `yaml:"enabled_dimensions" json:"enabled_dimensions"`
}

// DefaultDimensions defines the 4 concrete anchor dimensions enabled by default.
var DefaultDimensions = []string{
	DimensionObject,
	DimensionMotif,
	DimensionMood,
	DimensionConcept,
}

// AllDimensions defines all 4 supported anchor dimension IDs in standard order.
var AllDimensions = []string{
	DimensionObject,
	DimensionMotif,
	DimensionMood,
	DimensionConcept,
}

// DimensionAliases provides backward compatibility mapping from legacy dimensions to the 4 canonical anchor dimensions.
var DimensionAliases = map[string]string{
	// Legacy dimensions -> Canonical anchor dimensions
	"entity":            DimensionObject,
	"person_work":       DimensionObject,
	"event":             DimensionMotif,
	"symptom":           DimensionMotif,
	"emotion":           DimensionMood,
	"metaphor":          DimensionMood,
	"judgment":          DimensionConcept,
	"question":          DimensionConcept,
	"methodology":       DimensionConcept,
	"counter_intuitive": DimensionConcept,

	// Canonical identity mappings
	DimensionObject:  DimensionObject,
	DimensionMotif:   DimensionMotif,
	DimensionMood:    DimensionMood,
	DimensionConcept: DimensionConcept,
}

// NormalizeDimension maps any legacy or alias dimension to canonical anchor dimensions.
func NormalizeDimension(dim string) string {
	lower := strings.ToLower(strings.TrimSpace(dim))
	if canonical, ok := DimensionAliases[lower]; ok {
		return canonical
	}
	return dim
}

// BuiltinDimensionDefs registers the 4 concrete anchor dimension definitions.
var BuiltinDimensionDefs = map[string]DimensionDef{
	DimensionObject: {
		ID:            DimensionObject,
		Name:          "名物与通用指涉",
		Description:   "具体人/角色、作品、风物、工具、技术、食物、地点等名物；不仅包含专有名词(Claude, Colab)，也完全包含日常通用指涉如 菜市场, 叔叔, 外卖员, 猫, 设计师, 前端",
		GatePrompt:    "出现具体人/角色、作品、风物、工具、技术、食物、地点等名词或通用指涉",
		ExtractPrompt: "实心锚点（2~5个汉字或1~3个英文单词），紧凑名词/通用指涉，严禁超过8个字的长句解释",
	},
	DimensionMotif: {
		ID:            DimensionMotif,
		Name:          "处境与母题",
		Description:   "人类生活、对话与工作里反复上演的经典剧目、动作切片或行为场景(如 逃席, 吐槽模型, 创造需求, 薅羊毛, 体验新玩具, 额度焦虑, 熟人借钱, 调侃调休)",
		GatePrompt:    "存在人类生活、对话与工作里反复上演的经典剧目、动作切片、行为场景或生活处境",
		ExtractPrompt: "实心锚点（2~5个汉字或1~3个英文单词），动作切片与经典母题，严禁超过8个字的长句解释",
	},
	DimensionMood: {
		ID:            DimensionMood,
		Name:          "心境与心绪",
		Description:   "淬炼成名词的2~4字心境颗粒与审美体验(如 吐槽欲, 通透, 松弛感, 抽离感, 戏谑, 会心一笑, 烦躁, 踏实, 荒谬感)",
		GatePrompt:    "流露出鲜明的情绪颗粒、心境体验、审美感受或精神状态",
		ExtractPrompt: "实心锚点（2~4个汉字），淬炼成名词的心境颗粒，禁用临床病理词，严禁文学长句描述",
	},
	DimensionConcept: {
		ID:            DimensionConcept,
		Name:          "观念与议题",
		Description:   "思想交锋、价值观或判断背后的核心词条/论点抓手(如 伪提效, 创造新需求, 机械迎合, 以Bug改Bug, 前端即逻辑, 平台套路, 人情味)",
		GatePrompt:    "包含思想交锋、观点判断、反思批判或价值主张",
		ExtractPrompt: "实心锚点（2~5个汉字或1~3个英文单词），紧凑名词/概念议题抓手，严禁长句论点阐述",
	},
}

func init() {
	// Populate legacy keys pointing to canonical definitions for backward compatibility
	for legacyKey, canonicalKey := range DimensionAliases {
		if _, exists := BuiltinDimensionDefs[legacyKey]; !exists {
			if def, ok := BuiltinDimensionDefs[canonicalKey]; ok {
				legacyDef := def
				legacyDef.ID = legacyKey
				BuiltinDimensionDefs[legacyKey] = legacyDef
			}
		}
	}
}

// DefaultConfig returns default keyword engine configuration with 4 anchor dimensions enabled.
func DefaultConfig() Config {
	return Config{
		EnabledDimensions: append([]string{}, DefaultDimensions...),
	}
}

// GetActiveDimensions returns active DimensionDefs for the given configuration.
func (c *Config) GetActiveDimensions() []DimensionDef {
	enabled := c.EnabledDimensions
	if len(enabled) == 0 {
		enabled = DefaultDimensions
	}

	var defs []DimensionDef
	seen := make(map[string]bool)
	for _, rawID := range enabled {
		id := NormalizeDimension(rawID)
		if seen[id] {
			continue
		}
		if def, ok := BuiltinDimensionDefs[id]; ok {
			defs = append(defs, def)
			seen[id] = true
		}
	}
	return defs
}
