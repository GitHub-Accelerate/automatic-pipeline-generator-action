package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// writeWorkflowOutput writes the workflow to a file with proper formatting
func writeWorkflowOutput(workflow *GitHubWorkflow, destPath string) ([]byte, error) {
	// Marshal back to YAML with 2-space indentation
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(workflow); err != nil {
		return nil, fmt.Errorf("error marshaling workflow: %w", err)
	}
	encoder.Close()
	outputData := buf.Bytes()

	// Ensure destination directory exists
	if err := os.MkdirAll(".github/workflows", 0755); err != nil {
		return nil, fmt.Errorf("error creating workflows directory: %w", err)
	}

	// Write merged workflow
	if err := os.WriteFile(destPath, outputData, 0644); err != nil {
		return nil, fmt.Errorf("error writing destination file: %w", err)
	}

	return outputData, nil
}

// checkWorkflowChanged compares the current workflow with the new one
func checkWorkflowChanged(destPath string, newData []byte) bool {
	existingData, err := os.ReadFile(destPath)
	if err != nil {
		return true // File doesn't exist, so it's changed
	}
	return string(existingData) != string(newData)
}

// writeGitHubOutput writes the workflow-updated status to GITHUB_OUTPUT
func writeGitHubOutput(workflowChanged bool) {
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return
	}

	f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening GITHUB_OUTPUT: %v\n", err)
		return
	}
	defer f.Close()

	if workflowChanged {
		f.WriteString("workflow-updated=true\n")
		fmt.Println("Workflow was updated")
	} else {
		f.WriteString("workflow-updated=false\n")
		fmt.Println("No workflow changes detected")
	}
}

// modifyJobForMakefile modifies job steps to use Makefile targets if they exist
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
