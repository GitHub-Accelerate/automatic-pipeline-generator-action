package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// detectPythonProject checks if the current directory contains a Python project
func detectPythonProject() (bool, string) {
	// Check for common Python project indicators
	indicators := []string{
		"requirements.txt",
		"pyproject.toml",
		"setup.py",
		"Pipfile",
	}

	for _, indicator := range indicators {
		if _, err := os.Stat(indicator); err == nil {
			return true, indicator
		}
	}

	return false, ""
}

// loadPythonJobTemplate loads and processes the Python job template
func loadPythonJobTemplate(packagesToInstall, fetchDepth, languageVersion string) (string, *Job, error) {
	jobTemplatePath := "templates/python.yml"

	jobData, err := templatesFS.ReadFile(jobTemplatePath)
	if err != nil {
		return "", nil, fmt.Errorf("error reading job template: %w", err)
	}

	var jobWorkflow GitHubWorkflow
	if err := yaml.Unmarshal(jobData, &jobWorkflow); err != nil {
		return "", nil, fmt.Errorf("error parsing job template: %w", err)
	}

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

	applyFetchDepth(job, fetchDepth)
	applyPackagesToInstall(job, packagesToInstall)
	applyPythonLanguageVersion(job, languageVersion)
	detectAndConfigurePythonTools(job)
	addTrivySecuritySteps(job)

	return jobName, job, nil
}

// applyPythonLanguageVersion sets the Python version in the setup-python step
func applyPythonLanguageVersion(job *Job, languageVersion string) {
	if job == nil || languageVersion == "" {
		return
	}

	fmt.Printf("Setting Python language version to: %s\n", languageVersion)

	for _, step := range job.Steps {
		if step.Uses == "" || !strings.HasPrefix(step.Uses, "actions/setup-python@") {
			continue
		}

		if step.With == nil {
			step.With = make(map[string]interface{})
		}

		step.With["python-version"] = languageVersion
		return
	}
}

// detectAndConfigurePythonTools detects and configures Python package managers
func detectAndConfigurePythonTools(job *Job) {
	if job == nil {
		return
	}

	// Detect Poetry
	if isPoetryProject() {
		fmt.Println("Detected Poetry project, configuring Poetry workflow")
		configurePoetryWorkflow(job)
		return
	}

	// Detect uv
	if _, err := os.Stat("uv.lock"); err == nil {
		fmt.Println("Detected uv project, configuring uv workflow")
		configureUvWorkflow(job)
		return
	}

	// Default to pip (already configured in template)
	fmt.Println("Using pip workflow")
}

// isPoetryProject checks if pyproject.toml contains Poetry configuration
func isPoetryProject() bool {
	data, err := os.ReadFile("pyproject.toml")
	if err != nil {
		return false
	}

	content := string(data)
	return strings.Contains(content, "[tool.poetry]")
}

// configurePoetryWorkflow modifies the job to use Poetry instead of pip
func configurePoetryWorkflow(job *Job) {
	// Update cache type
	for _, step := range job.Steps {
		if step.Uses != "" && strings.HasPrefix(step.Uses, "actions/setup-python@") {
			if step.With != nil {
				step.With["cache"] = "poetry"
			}
		}
	}

	// Replace install dependencies step
	for _, step := range job.Steps {
		if step.Name == "Install dependencies" {
			step.Run = "poetry install"
		}
		if step.Name == "Install ruff" {
			step.Run = "poetry add --group dev ruff"
		}
		if step.Name == "Run tests" {
			step.Run = "poetry run pytest"
		}
		if step.Name == "Build package" {
			step.Run = "poetry build"
		}
	}
}

// configureUvWorkflow modifies the job to use uv instead of pip
func configureUvWorkflow(job *Job) {
	// Update cache - uv doesn't have built-in cache yet, keep pip
	for _, step := range job.Steps {
		if step.Uses != "" && strings.HasPrefix(step.Uses, "actions/setup-python@") {
			if step.With != nil {
				// uv can work with pip cache
				step.With["cache"] = "pip"
			}
		}
	}

	// Add uv installation step after Python setup
	uvSetupStep := &Step{
		Name: "Install uv",
		Run:  "pip install uv",
	}

	var newSteps []*Step
	for _, step := range job.Steps {
		newSteps = append(newSteps, step)
		if step.Uses != "" && strings.HasPrefix(step.Uses, "actions/setup-python@") {
			newSteps = append(newSteps, uvSetupStep)
		}
	}
	job.Steps = newSteps

	// Replace install dependencies step
	for _, step := range job.Steps {
		if step.Name == "Install dependencies" {
			step.Run = "uv pip install -r requirements.txt"
		}
		if step.Name == "Install ruff" {
			step.Run = "uv pip install ruff"
		}
		if step.Name == "Build package" {
			step.Run = `uv pip install build
python -m build`
		}
	}
}
