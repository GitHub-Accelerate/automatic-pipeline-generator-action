package main

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// generateWorkflowData marshals the workflow to YAML bytes with proper formatting
func generateWorkflowData(workflow *GitHubWorkflow) ([]byte, error) {
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(workflow); err != nil {
		return nil, fmt.Errorf("error marshaling workflow: %w", err)
	}
	encoder.Close()
	return buf.Bytes(), nil
}

// writeWorkflowToFile writes the workflow data to a file
func writeWorkflowToFile(destPath string, outputData []byte) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(".github/workflows", 0755); err != nil {
		return fmt.Errorf("error creating workflows directory: %w", err)
	}

	// Write merged workflow
	if err := os.WriteFile(destPath, outputData, 0644); err != nil {
		return fmt.Errorf("error writing destination file: %w", err)
	}

	return nil
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
