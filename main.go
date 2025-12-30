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
	var packagesToInstall, languageVersion, fetchDepth string

	if len(os.Args) > 1 {
		packagesToInstall = os.Args[1]
	}
	// Skip dockerImageToUse (os.Args[2]) - not yet implemented
	if len(os.Args) > 3 {
		languageVersion = os.Args[3]
	}
	// Skip itemToBuild (os.Args[4) - not yet implemented
	if len(os.Args) > 5 {
		fetchDepth = os.Args[5]
	}

	// Detect project language and get job template
	var jobName string
	var job *Job
	var err error

	if detected, indicator := detectPythonProject(); detected {
		fmt.Printf("Detected Python project (indicator: %s)\n", indicator)
		jobName, job, err = loadPythonJobTemplate(packagesToInstall, fetchDepth, languageVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else if detected, indicator := detectGoProject(); detected {
		fmt.Printf("Detected Go project (indicator: %s)\n", indicator)
		jobName, job, err = loadGoJobTemplate(packagesToInstall, fetchDepth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else if detected, indicator := detectJavaMavenProject(); detected {
		fmt.Printf("Detected Java Maven project (indicator: %s)\n", indicator)
		jobName, job, err = loadJavaMavenJobTemplate(packagesToInstall, fetchDepth, languageVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else if detected, indicator := detectJavaGradleProject(); detected {
		fmt.Printf("Detected Java Gradle project (indicator: %s)\n", indicator)
		jobName, job, err = loadJavaGradleJobTemplate(packagesToInstall, fetchDepth, languageVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else if detected, indicator := detectCSharpProject(); detected {
		fmt.Printf("Detected C# project (indicator: %s)\n", indicator)
		jobName, job, err = loadCSharpJobTemplate(packagesToInstall, fetchDepth, languageVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else if detected, indicator := detectCCppProject(); detected {
		fmt.Printf("Detected C/C++ project (indicator: %s)\n", indicator)
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

	// Apply fetch_depth to the run-generator job if specified
	if fetchDepth != "" && baseWorkflow.Jobs["run-generator"] != nil {
		fmt.Println("Applying fetch depth to run-generator job")
		applyFetchDepth(baseWorkflow.Jobs["run-generator"], fetchDepth)
	}

	// Apply input parameters to the run-generator job's action step
	if baseWorkflow.Jobs["run-generator"] != nil {
		applyActionInputs(baseWorkflow.Jobs["run-generator"], packagesToInstall, fetchDepth)
	}

	// Build the final workflow
	workflow := buildWorkflow(baseWorkflow, jobName, job)

	// Generate output data first (before writing to file)
	outputData, err := generateWorkflowData(&workflow)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Check if workflow changed BEFORE writing
	workflowChanged := checkWorkflowChanged(destPath, outputData)

	// Write workflow to file
	if err := writeWorkflowToFile(destPath, outputData); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully merged job '%s' into %s\n", jobName, destPath)

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
