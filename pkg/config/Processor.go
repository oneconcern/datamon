package config

// Spec for what to process
type Processor struct {
	Name      string    `json:"name" yaml:"name"`
	Branch    string    `json:"branch" yaml:"branch"`
	Runtime   string    `json:"runtime" yaml:"runtime"`
	Resources Resources `json:"resources" yaml:"resources"`
	Content   []string  `json:"content" yaml:"content"`
	Command   []string  `json:"command" yaml:"command"`
}

// Resources for the processor
type Resources struct {
	CPU ResourceLimit `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Mem ResourceLimit `json:"mem,omitempty" yaml:"mem,omitempty"`
}

// Resource limits
type ResourceLimit struct {
	Min string `json:"min,omitempty" yaml:"min,omitempty"`
	Max string `json:"max,omitempty" yaml:"max,omitempty"`
}
