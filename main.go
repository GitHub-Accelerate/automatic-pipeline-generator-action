package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*
var templatesFS embed.FS

func modifyJobForMakefile(job *Job) {
	// Parse Makefile to check for build and test targets
	makefileData, err := os.ReadFile("Makefile")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading Makefile: %v\n", err)
		return
	}

	makefileContent := string(makefileData)
	hasBuildTarget := false
	hasTestTarget := false

	// Simple parsing: look for "build:" or "test:" at the start of lines
	lines := strings.Split(makefileContent, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "build:") {
			hasBuildTarget = true
		}
		if strings.HasPrefix(trimmed, "test:") {
			hasTestTarget = true
		}
	}

	if !hasBuildTarget && !hasTestTarget {
		fmt.Println("No 'build' or 'test' targets found in Makefile, replacement aborted as it would fail to run")
		return
	}

	// Modify the job steps to use make commands
	for _, step := range job.Steps {
		if step.Name == "Build" && hasBuildTarget {
			step.Run = "make build"
			fmt.Println("Replaced Build step with 'make build'")
		} else if step.Name == "Test" && hasTestTarget {
			step.Run = "make test"
			fmt.Println("Replaced Test step with 'make test'")
		}
	}

	if !hasBuildTarget {
		fmt.Println("No 'build' target found in Makefile, Build step not replaced")
	}
	if !hasTestTarget {
		fmt.Println("No 'test' target found in Makefile, Test step not replaced")
	}
}

func main() {
	fmt.Printf("Number of arguments: %d\n", len(os.Args))
	fmt.Println("Arguments:", os.Args)

	// Check if go.mod exists in current directory
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("No go.mod found in current directory")
		os.Exit(1)
	}

	fmt.Println("Found go.mod file")

	// Define paths
	generatorTemplatePath := "templates/generator.yml"
	jobTemplatePath := "templates/go.yml"
	destPath := ".github/workflows/main.yml"

	// Load the base generator template (contains workflow metadata and run-generator job)
	generatorData, err := templatesFS.ReadFile(generatorTemplatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading generator template: %v\n", err)
		os.Exit(1)
	}

	var baseWorkflow GitHubWorkflow
	if err := yaml.Unmarshal(generatorData, &baseWorkflow); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing generator template: %v\n", err)
		os.Exit(1)
	}

	// Load job template from embedded filesystem
	jobData, err := templatesFS.ReadFile(jobTemplatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading job template: %v\n", err)
		os.Exit(1)
	}

	var jobWorkflow GitHubWorkflow
	if err := yaml.Unmarshal(jobData, &jobWorkflow); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing job template: %v\n", err)
		os.Exit(1)
	}

	// Get the job from job template
	var sourceJobName string
	var sourceJob *Job
	for name, job := range jobWorkflow.Jobs {
		sourceJobName = name
		sourceJob = job
		break
	}

	if sourceJob == nil {
		fmt.Fprintf(os.Stderr, "No jobs found in job template\n")
		os.Exit(1)
	}

	// Check if Makefile exists and modify build/test steps accordingly
	if _, err := os.Stat("Makefile"); err == nil {
		fmt.Println("Found Makefile, updating build and test commands")
		modifyJobForMakefile(sourceJob)
	}

	// Start with the base workflow structure (enforces name, on, permissions, run-generator job)
	destWorkflow := baseWorkflow

	// Ensure jobs map exists
	if destWorkflow.Jobs == nil {
		destWorkflow.Jobs = make(map[string]*Job)
	}

	// Ensure run-generator is first in job order
	if len(destWorkflow.jobOrder) == 0 || destWorkflow.jobOrder[0] != "run-generator" {
		// Reset job order with run-generator first
		destWorkflow.jobOrder = []string{"run-generator"}
	}

	// Add the source job after run-generator
	if sourceJobName != "run-generator" {
		// Check if job already exists in order
		jobExists := false
		for _, name := range destWorkflow.jobOrder {
			if name == sourceJobName {
				jobExists = true
				break
			}
		}
		if !jobExists {
			destWorkflow.jobOrder = append(destWorkflow.jobOrder, sourceJobName)
		}
		destWorkflow.Jobs[sourceJobName] = sourceJob
	}

	// Marshal back to YAML with 2-space indentation
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&destWorkflow); err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling workflow: %v\n", err)
		os.Exit(1)
	}
	encoder.Close()
	outputData := buf.Bytes()

	// Ensure destination directory exists
	if err := os.MkdirAll(".github/workflows", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workflows directory: %v\n", err)
		os.Exit(1)
	}

	// Check if workflow changed by comparing content
	workflowChanged := true
	if existingData, err := os.ReadFile(destPath); err == nil {
		if string(existingData) == string(outputData) {
			workflowChanged = false
		}
	}

	// Write merged workflow
	if err := os.WriteFile(destPath, outputData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing destination file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully merged job '%s' into %s\n", sourceJobName, destPath)

	// If workflow changed, commit and push
	if workflowChanged {
		if err := commitAndPushWorkflow(destPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error committing and pushing workflow: %v\n", err)
			os.Exit(1)
		}
	}

	// Write to GITHUB_OUTPUT
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput != "" {
		f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening GITHUB_OUTPUT: %v\n", err)
		} else {
			defer f.Close()
			if workflowChanged {
				f.WriteString("workflow-updated=true\n")
				fmt.Println("Workflow was updated")
			} else {
				f.WriteString("workflow-updated=false\n")
				fmt.Println("No workflow changes detected")
			}
		}
	}
}
