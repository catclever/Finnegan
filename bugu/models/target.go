package models

type Target struct {
	ID               string   `yaml:"id" json:"id"`
	Type             string   `yaml:"type" json:"type"`
	Value            string   `yaml:"value" json:"value"`
	Languages        []string `yaml:"languages" json:"languages"`
	Tags             []string `yaml:"tags" json:"tags"`
	PreferredEngines []string `yaml:"preferred_engines" json:"preferred_engines,omitempty"`
}

type RadarTargets struct {
	Targets []Target `yaml:"targets"`
}

type LLMSearchPlan struct {
	SelectedTargetIDs []string            `json:"selected_target_ids"`
	SearchQueries     map[string][]string `json:"search_queries"`
}
