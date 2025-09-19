package routing

// Rule represents a routing rule
type Rule struct {
	Name           string     `json:"name" yaml:"name"`
	Description    string     `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled        bool       `json:"enabled" yaml:"enabled"`
	Priority       int        `json:"priority" yaml:"priority"`
	Conditions     Conditions `json:"conditions" yaml:"conditions"`
	Actions        Actions    `json:"actions" yaml:"actions"`
	StopProcessing bool       `json:"stop_processing,omitempty" yaml:"stop_processing,omitempty"`
}

// Conditions define when a rule should be applied
type Conditions struct {
	Priorities []int             `json:"priorities,omitempty" yaml:"priorities,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Template   string            `json:"template,omitempty" yaml:"template,omitempty"`
	Platform   string            `json:"platform,omitempty" yaml:"platform,omitempty"`
}

// Actions define what should happen when a rule matches
type Actions struct {
	Targets     []Target          `json:"targets,omitempty" yaml:"targets,omitempty"`
	AddMetadata map[string]string `json:"add_metadata,omitempty" yaml:"add_metadata,omitempty"`
	SetPriority int               `json:"set_priority,omitempty" yaml:"set_priority,omitempty"`
	SetPlatform string            `json:"set_platform,omitempty" yaml:"set_platform,omitempty"`
}

// Target represents a routing target (separate from notifier target to avoid dependency)
type Target struct {
	Type     string            `json:"type" yaml:"type"`
	Value    string            `json:"value" yaml:"value"`
	Platform string            `json:"platform" yaml:"platform"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
