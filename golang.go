package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// detectGoProject checks if the current directory contains a Go project
func detectGoProject() bool {
	_, err := os.Stat("go.mod")
	return err == nil
}

// loadGoJobTemplate loads and processes the Go job template
func loadGoJobTemplate() (string, *Job, error) {
	jobTemplatePath := "templates/go.yml"

	// Load job template from embedded filesystem
	jobData, err := templatesFS.ReadFile(jobTemplatePath)
	if err != nil {
		return "", nil, fmt.Errorf("error reading job template: %w", err)
	}

	var jobWorkflow GitHubWorkflow
	if err := yaml.Unmarshal(jobData, &jobWorkflow); err != nil {
		return "", nil, fmt.Errorf("error parsing job template: %w", err)
	}

	// Get the job from job template
	var jobName string
	var job *Job
	for name, j := range jobWorkflow.Jobs {
		jobName = name
		job = j
		break
	}

	if job == nil {
		return "", nil, fmt.Errorf("no jobs found in job template")
	}

	// Check if Makefile exists and modify build/test steps accordingly
	if _, err := os.Stat("Makefile"); err == nil {
		fmt.Println("Found Makefile, updating build and test commands")
		modifyJobForMakefile(job)
	}

	return jobName, job, nil
}
