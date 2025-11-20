package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// detectCCppProject checks if the current directory contains a C/C++ project
func detectCCppProject() bool {
	// Check for common C/C++ indicators
	indicators := []string{
		"CMakeLists.txt",
		"configure.ac",
		"meson.build",
		".c",
		".cpp",
		".cc",
		".h",
		".hpp",
		".clang-format",
	}

	for _, indicator := range indicators {
		if _, err := os.Stat(indicator); err == nil {
			return true
		}
	}

	// Check if directory contains .c or .cpp files
	entries, err := os.ReadDir(".")
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".c") || strings.HasSuffix(name, ".cpp") ||
			strings.HasSuffix(name, ".cc") || strings.HasSuffix(name, ".cxx") {
			return true
		}
	}

	return false
}

// loadCCppJobTemplate loads and processes the C/C++ job template
func loadCCppJobTemplate(packagesToInstall, fetchDepth string) (string, *Job, error) {
	jobTemplatePath := "templates/c_cpp.yml"

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

	// Apply customizations
	applyFetchDepth(job, fetchDepth)
	applyPackagesToInstall(job, packagesToInstall)

	// Check if Makefile exists and modify build/test steps accordingly
	if _, err := os.Stat("Makefile"); err == nil {
		fmt.Println("Found Makefile, updating build and test commands")
		modifyJobForMakefile(job)
	}

	return jobName, job, nil
}
