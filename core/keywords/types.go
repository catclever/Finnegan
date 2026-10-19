package keywords

// 10 Cognitive Dimension Constants
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

// DefaultDimensions defines the 6 dimensions enabled by default.
var DefaultDimensions = []string{
	DimensionEntity,
	DimensionEvent,
	DimensionJudgment,
	DimensionEmotion,
	DimensionQuestion,
	DimensionMetaphor,
}

// AllDimensions defines all 10 supported dimension IDs in standard order.
var AllDimensions = []string{
	DimensionEntity,
	DimensionEvent,
	DimensionJudgment,
	DimensionEmotion,
	DimensionQuestion,
	DimensionMetaphor,
	DimensionSymptom,
	DimensionMethodology,
	DimensionCounterIntuitive,
	DimensionPersonWork,
}

// BuiltinDimensionDefs registers the 10 built-in cognitive dimension definitions.
var BuiltinDimensionDefs = map[string]DimensionDef{
	DimensionEntity: {
		ID:            DimensionEntity,
		Name:          "专用名词 / 实体",
		Description:   "技术栈、协议、软件、学术概念或专有黑话",
		GatePrompt:    "出现特定技术栈、协议、软件、学术概念、特定流派或专有黑话",
		ExtractPrompt: "原教旨高保真：保留精确术语、版本号、原名（如 WebKitGTK 4.1），严禁上位词泛化",
	},
	DimensionEvent: {
		ID:            DimensionEvent,
		Name:          "事件 / 动态",
		Description:   "正在发生的行业动态、版本发布、社会事件或具体变动",
		GatePrompt:    "提及正在发生的行业动态、版本发布、社会/圈层事件或具体变动",
		ExtractPrompt: "主体+动态锚定：提取“谁/什么 + 发生了什么变化”的紧凑短语",
	},
	DimensionJudgment: {
		ID:            DimensionJudgment,
		Name:          "判断 / 命题",
		Description:   "鲜明的观点、价值判断、批判或二元对立的思考张力",
		GatePrompt:    "蕴含鲜明的观点、价值判断、批判、或者二元对立的思考张力",
		ExtractPrompt: "张力命题化：转化为可激发深度辩论的思辨命题或反常识切口（如 Local-First 的同步悖论）",
	},
	DimensionEmotion: {
		ID:            DimensionEmotion,
		Name:          "情绪 / 心境",
		Description:   "情感基调、生存境遇体验、深夜独白或心境状态",
		GatePrompt:    "流露出明显的情感基调、生存境遇体验、深夜独白或心境状态",
		ExtractPrompt: "去临床化切片：禁用“焦虑/抑郁”等医学词，提取具象生活与心境短语（如 深夜独自排错的宁静感）",
	},
	DimensionQuestion: {
		ID:            DimensionQuestion,
		Name:          "待解问题 / 求索",
		Description:   "明确的困惑、悬而未决的疑问、方案选型纠结或求助意图",
		GatePrompt:    "存在明确的困惑、悬而未决的疑问、方案选型纠结或求助意图",
		ExtractPrompt: "How/Why 问题重构：提炼为社区高信息密度的提问式短语",
	},
	DimensionMetaphor: {
		ID:            DimensionMetaphor,
		Name:          "隐喻 / 意象",
		Description:   "文学修辞、通感、哲学意象、梦境碎片或美学符号",
		GatePrompt:    "包含独特的文学修辞、通感、哲学意象、梦境碎片或美学符号",
		ExtractPrompt: "意象结晶：保留核心隐喻体与美学意象组合（如 数字废墟中的苔藓、赛博招魂）",
	},
	DimensionSymptom: {
		ID:            DimensionSymptom,
		Name:          "异常 / 踩坑现象",
		Description:   "具体 Bug 表现、诡异报错、性能劣化或反常物理/系统现象",
		GatePrompt:    "描述具体的 Bug 表现、诡异报错、性能劣化或反常物理/系统现象",
		ExtractPrompt: "高区分度特征：提取核心错误特征码、环境条件与异常表现",
	},
	DimensionMethodology: {
		ID:            DimensionMethodology,
		Name:          "方法论 / 范式",
		Description:   "工作流、思维模型、架构模式、效率实践或做事法则",
		GatePrompt:    "探讨工作流、思维模型、架构模式、效率实践或做事法则",
		ExtractPrompt: "范式提炼：提取为“X 驱动的 Y”、“Z 工作流/原则”结构",
	},
	DimensionCounterIntuitive: {
		ID:            DimensionCounterIntuitive,
		Name:          "反直觉 / 悖论",
		Description:   "常识背后的荒谬、事与愿违的规律或讽刺现象",
		GatePrompt:    "揭示某种常识背后的荒谬、事与愿违的规律或讽刺现象",
		ExtractPrompt: "悖论对比式：提取出“表面 A 实则 B”的核心冲突词",
	},
	DimensionPersonWork: {
		ID:            DimensionPersonWork,
		Name:          "人物 / 作品坐标",
		Description:   "特定作者、思想家、导演、书籍、论文、电影、播客或开源作品",
		GatePrompt:    "提及特定作者、思想家、导演、书籍、论文、电影、播客或开源作品",
		ExtractPrompt: "精确书名/人名对齐：提取标准译名与原名对",
	},
}

// DefaultConfig returns default keyword engine configuration with 6 default dimensions enabled.
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
	for _, id := range enabled {
		if def, ok := BuiltinDimensionDefs[id]; ok {
			defs = append(defs, def)
		}
	}
	return defs
}
