package main

import (
	"embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var templatesFS embed.FS

func main() {
	fmt.Printf("Number of arguments: %d\n", len(os.Args))
	fmt.Println("Arguments:", os.Args)

	// Parse command line arguments (from action inputs)
	var packagesToInstall, fetchDepth string

	if len(os.Args) > 1 {
		packagesToInstall = os.Args[1]
	}
	// Skip dockerImageToUse (os.Args[2]) - not yet implemented
	// Skip langageVersion (os.Args[3]) - not yet implemented
	// Skip itemToBuild (os.Args[4) - not yet implemented
	if len(os.Args) > 5 {
		fetchDepth = os.Args[5]
	}

	// Detect project language and get job template
	var jobName string
	var job *Job
	var err error

	if detectGoProject() {
		fmt.Println("Detected Go project")
		jobName, job, err = loadGoJobTemplate(packagesToInstall, fetchDepth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else if detectCCppProject() {
		fmt.Println("Detected C/C++ project")
		jobName, job, err = loadCCppJobTemplate(packagesToInstall, fetchDepth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("No supported project type found in current directory")
		os.Exit(1)
	}

	// Define paths
	generatorTemplatePath := "templates/generator.yml"
	destPath := ".github/workflows/main.yml"

	// Load the base generator template (contains workflow metadata and run-generator job)
	baseWorkflow, err := loadBaseWorkflow(generatorTemplatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Build the final workflow
	workflow := buildWorkflow(baseWorkflow, jobName, job)

	// Write workflow to file
	outputData, err := writeWorkflowOutput(&workflow, destPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully merged job '%s' into %s\n", jobName, destPath)

	// Check if workflow changed
	workflowChanged := checkWorkflowChanged(destPath, outputData)

	// If workflow changed, commit and push
	if workflowChanged {
		if err := commitAndPushWorkflow(destPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error committing and pushing workflow: %v\n", err)
			os.Exit(1)
		}
	}

	// Write to GITHUB_OUTPUT
	writeGitHubOutput(workflowChanged)
}

// loadBaseWorkflow loads the base workflow template with standard structure
func loadBaseWorkflow(templatePath string) (GitHubWorkflow, error) {
	generatorData, err := templatesFS.ReadFile(templatePath)
	if err != nil {
		return GitHubWorkflow{}, fmt.Errorf("error reading generator template: %w", err)
	}

	var baseWorkflow GitHubWorkflow
	if err := yaml.Unmarshal(generatorData, &baseWorkflow); err != nil {
		return GitHubWorkflow{}, fmt.Errorf("error parsing generator template: %w", err)
	}

	return baseWorkflow, nil
}

// buildWorkflow constructs the final workflow with the base structure and project-specific job
func buildWorkflow(baseWorkflow GitHubWorkflow, jobName string, job *Job) GitHubWorkflow {
	// Start with the base workflow structure (enforces name, on, permissions, run-generator job)
	workflow := baseWorkflow

	// Ensure jobs map exists
	if workflow.Jobs == nil {
		workflow.Jobs = make(map[string]*Job)
	}

	// Ensure run-generator is first in job order
	if len(workflow.jobOrder) == 0 || workflow.jobOrder[0] != "run-generator" {
		// Reset job order with run-generator first
		workflow.jobOrder = []string{"run-generator"}
	}

	// Add the source job after run-generator
	if jobName != "run-generator" {
		// Check if job already exists in order
		jobExists := false
		for _, name := range workflow.jobOrder {
			if name == jobName {
				jobExists = true
				break
			}
		}
		if !jobExists {
			workflow.jobOrder = append(workflow.jobOrder, jobName)
		}
		workflow.Jobs[jobName] = job
	}

	return workflow
}
