package main

import (
	"gopkg.in/yaml.v3"
)

// GitHubWorkflow represents a complete GitHub Actions workflow
type GitHubWorkflow struct {
	Name        string          `yaml:"name,omitempty"`
	On          interface{}     `yaml:"on,omitempty"`          // Can be string, array, or map
	Permissions interface{}     `yaml:"permissions,omitempty"` // Can be string or map
	Jobs        map[string]*Job `yaml:"jobs"`
	jobOrder    []string        // Internal field to track job order
}

// Job represents a single job in a GitHub Actions workflow
type Job struct {
	Name      string                 `yaml:"name,omitempty"`
	RunsOn    interface{}            `yaml:"runs-on"`         // Can be string or array
	Needs     interface{}            `yaml:"needs,omitempty"` // Can be string or array
	If        string                 `yaml:"if,omitempty"`
	Outputs   map[string]string      `yaml:"outputs,omitempty"`
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

// UnmarshalYAML implements custom YAML unmarshaling to preserve job order
func (w *GitHubWorkflow) UnmarshalYAML(node *yaml.Node) error {
	// Create a temporary struct with all fields
	type workflowAlias GitHubWorkflow
	temp := (*workflowAlias)(w)

	// Unmarshal normally
	if err := node.Decode(temp); err != nil {
		return err
	}

	// Extract job order from the YAML node
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "jobs" {
			jobsNode := node.Content[i+1]
			w.jobOrder = make([]string, 0, len(jobsNode.Content)/2)
			for j := 0; j < len(jobsNode.Content); j += 2 {
				w.jobOrder = append(w.jobOrder, jobsNode.Content[j].Value)
			}
			break
		}
	}

	return nil
}

// MarshalYAML implements custom YAML marshaling to preserve field order
func (w *GitHubWorkflow) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	// Add fields in desired order
	if w.Name != "" {
		addToNode(node, "name", w.Name)
	}
	if w.On != nil {
		addToNode(node, "on", w.On)
	}
	if w.Permissions != nil {
		addToNode(node, "permissions", w.Permissions)
	}
	if w.Jobs != nil {
		// Create jobs node with preserved order
		jobsNode := &yaml.Node{Kind: yaml.MappingNode}

		// Add jobs in the original order
		if len(w.jobOrder) > 0 {
			for _, jobName := range w.jobOrder {
				if job, exists := w.Jobs[jobName]; exists {
					addToNode(jobsNode, jobName, job)
				}
			}
		} else {
			// Fallback: add jobs in arbitrary order if jobOrder not set
			for jobName, job := range w.Jobs {
				addToNode(jobsNode, jobName, job)
			}
		}

		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "jobs"}
		node.Content = append(node.Content, keyNode, jobsNode)
	}

	return node, nil
}

// MarshalYAML implements custom YAML marshaling to preserve field order
func (j *Job) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	// Add fields in desired order
	if j.Name != "" {
		addToNode(node, "name", j.Name)
	}
	if j.RunsOn != nil {
		addToNode(node, "runs-on", j.RunsOn)
	}
	if j.Needs != nil {
		addToNode(node, "needs", j.Needs)
	}
	if j.If != "" {
		addToNode(node, "if", j.If)
	}
	if j.Outputs != nil {
		addToNode(node, "outputs", j.Outputs)
	}
	if j.Env != nil {
		addToNode(node, "env", j.Env)
	}
	if j.Steps != nil {
		addToNode(node, "steps", j.Steps)
	}
	if j.Strategy != nil {
		addToNode(node, "strategy", j.Strategy)
	}
	if j.Container != nil {
		addToNode(node, "container", j.Container)
	}
	if j.Services != nil {
		addToNode(node, "services", j.Services)
	}

	return node, nil
}

// MarshalYAML implements custom YAML marshaling to preserve field order
func (s *Step) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	// Add fields in desired order
	if s.Name != "" {
		addToNode(node, "name", s.Name)
	}
	if s.ID != "" {
		addToNode(node, "id", s.ID)
	}
	if s.Uses != "" {
		addToNode(node, "uses", s.Uses)
	}
	if s.Run != "" {
		addToNode(node, "run", s.Run)
	}
	if s.With != nil {
		addToNode(node, "with", s.With)
	}
	if s.Env != nil {
		addToNode(node, "env", s.Env)
	}
	if s.If != "" {
		addToNode(node, "if", s.If)
	}

	return node, nil
}

// Helper function to add key-value pairs to a YAML node
func addToNode(node *yaml.Node, key string, value interface{}) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valueNode := &yaml.Node{}
	valueNode.Encode(value)
	node.Content = append(node.Content, keyNode, valueNode)
}
