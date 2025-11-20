package main

// GitHubWorkflow represents a complete GitHub Actions workflow
type GitHubWorkflow struct {
	Name string          `yaml:"name,omitempty"`
	On   interface{}     `yaml:"on,omitempty"` // Can be string, array, or map
	Jobs map[string]*Job `yaml:"jobs"`
}

// Job represents a single job in a GitHub Actions workflow
type Job struct {
	Name      string                 `yaml:"name,omitempty"`
	RunsOn    interface{}            `yaml:"runs-on"`         // Can be string or array
	Needs     interface{}            `yaml:"needs,omitempty"` // Can be string or array
	If        string                 `yaml:"if,omitempty"`
	Env       map[string]string      `yaml:"env,omitempty"`
	Steps     []*Step                `yaml:"steps"`
	Strategy  *Strategy              `yaml:"strategy,omitempty"`
	Container interface{}            `yaml:"container,omitempty"`
	Services  map[string]interface{} `yaml:"services,omitempty"`
}

// Step represents a single step in a job
type Step struct {
	Name string            `yaml:"name,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	Run  string            `yaml:"run,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
	If   string            `yaml:"if,omitempty"`
	ID   string            `yaml:"id,omitempty"`
}

// Strategy represents job strategy configuration
type Strategy struct {
	Matrix      map[string]interface{} `yaml:"matrix,omitempty"`
	FailFast    *bool                  `yaml:"fail-fast,omitempty"`
	MaxParallel int                    `yaml:"max-parallel,omitempty"`
}
