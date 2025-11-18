package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Printf("Number of arguments: %d\n", len(os.Args))
	fmt.Println("Arguments:", os.Args)

	// Check if go.mod exists in current directory
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("No go.mod found in current directory")
		os.Exit(1)
	}

	fmt.Println("Found go.mod file")

	// Define source and destination paths
	sourcePath := "templates/go.yml"
	destPath := ".github/workflows/main.yml"

	// Load source template
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source template: %v\n", err)
		os.Exit(1)
	}

	var sourceWorkflow map[string]interface{}
	if err := yaml.Unmarshal(sourceData, &sourceWorkflow); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing source template: %v\n", err)
		os.Exit(1)
	}

	// Extract the job from source workflow
	sourceJobs, ok := sourceWorkflow["jobs"].(map[string]interface{})
	if !ok || len(sourceJobs) == 0 {
		fmt.Fprintf(os.Stderr, "No jobs found in source template\n")
		os.Exit(1)
	}

	// Get the first (and should be only) job
	var sourceJobName string
	var sourceJob interface{}
	for name, job := range sourceJobs {
		sourceJobName = name
		sourceJob = job
		break
	}

	// Check if Makefile exists and modify build/test steps accordingly
	if _, err := os.Stat("Makefile"); err == nil {
		fmt.Println("Found Makefile, updating build and test commands")

		// Modify the job steps to use make commands
		if jobMap, ok := sourceJob.(map[string]interface{}); ok {
			if steps, ok := jobMap["steps"].([]interface{}); ok {
				for _, step := range steps {
					if stepMap, ok := step.(map[string]interface{}); ok {
						if name, ok := stepMap["name"].(string); ok {
							if name == "Build" {
								stepMap["run"] = "make build"
							} else if name == "Test" {
								stepMap["run"] = "make test"
							}
						}
					}
				}
			}
		}
	}

	// Load or create destination workflow
	var destWorkflow map[string]interface{}
	if destData, err := os.ReadFile(destPath); err == nil {
		if err := yaml.Unmarshal(destData, &destWorkflow); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing destination workflow: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Create new workflow structure
		destWorkflow = make(map[string]interface{})
	}

	// Ensure jobs section exists
	if destWorkflow["jobs"] == nil {
		destWorkflow["jobs"] = make(map[string]interface{})
	}
	destJobs := destWorkflow["jobs"].(map[string]interface{})

	// Add the job to destination workflow
	destJobs[sourceJobName] = sourceJob

	// Marshal back to YAML
	outputData, err := yaml.Marshal(destWorkflow)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling workflow: %v\n", err)
		os.Exit(1)
	}

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
