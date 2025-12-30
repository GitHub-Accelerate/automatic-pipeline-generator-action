package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// detectCSharpProject checks if the current directory contains a C# project
func detectCSharpProject() (bool, string) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return false, ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sln") {
			return true, entry.Name()
		}
	}

	return false, ""
}

// loadCSharpJobTemplate loads and processes the C# job template
func loadCSharpJobTemplate(packagesToInstall, fetchDepth, languageVersion string) (string, *Job, error) {
	jobTemplatePath := "templates/csharp.yml"

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
	applyDotNetVersion(job, languageVersion)
	configureCSharpProjects(job)

	return jobName, job, nil
}

// applyDotNetVersion sets the .NET version in the setup-dotnet step
func applyDotNetVersion(job *Job, languageVersion string) {
	if job == nil || languageVersion == "" {
		return
	}

	fmt.Printf("Setting .NET version to: %s\n", languageVersion)

	for _, step := range job.Steps {
		if step.Uses == "" || !strings.HasPrefix(step.Uses, "actions/setup-dotnet@") {
			continue
		}

		if step.With == nil {
			step.With = make(map[string]interface{})
		}

		step.With["dotnet-version"] = languageVersion
		return
	}
}

// configureCSharpProjects parses .sln file and configures build steps for .csproj files
func configureCSharpProjects(job *Job) {
	if job == nil {
		return
	}

	slnFile := findSlnFile()
	if slnFile == "" {
		return
	}

	csprojFiles := parseSolutionFile(slnFile)
	if len(csprojFiles) == 0 {
		fmt.Println("No .csproj files found in solution, using default build")
		return
	}

	fmt.Printf("Found %d .csproj file(s) in solution: %v\n", len(csprojFiles), csprojFiles)

	// Update build and test commands to target specific projects
	buildCommand := "dotnet build --no-restore --configuration Release"
	testCommand := "dotnet test --no-build --configuration Release --verbosity normal"
	publishCommand := "dotnet publish --no-build --configuration Release --output ./publish"

	for _, step := range job.Steps {
		switch step.Name {
		case "Build":
			if len(csprojFiles) == 1 {
				step.Run = fmt.Sprintf("%s %s", buildCommand, csprojFiles[0])
			} else {
				step.Run = buildCommand
			}
		case "Test":
			step.Run = testCommand
		case "Publish":
			if len(csprojFiles) == 1 {
				step.Run = fmt.Sprintf("%s %s", publishCommand, csprojFiles[0])
			} else {
				step.Run = publishCommand
			}
		}
	}
}

// findSlnFile finds the first .sln file in the current directory
func findSlnFile() string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sln") {
			return entry.Name()
		}
	}

	return ""
}

// parseSolutionFile parses a .sln file and extracts .csproj project paths
func parseSolutionFile(slnPath string) []string {
	file, err := os.Open(slnPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening solution file %s: %v\n", slnPath, err)
		return nil
	}
	defer file.Close()

	var csprojFiles []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for Project lines in the solution file
		// Format: Project("{...}") = "ProjectName", "path\to\project.csproj", "{...}"
		if strings.HasPrefix(line, "Project(") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 5 {
				projectPath := parts[3]
				if strings.HasSuffix(projectPath, ".csproj") {
					// Convert Windows path separators to Unix
					projectPath = strings.ReplaceAll(projectPath, "\\", "/")
					csprojFiles = append(csprojFiles, projectPath)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading solution file %s: %v\n", slnPath, err)
		return nil
	}

	return csprojFiles
}
