package main

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"gopkg.in/yaml.v3"
)

func modifyJobForMakefile(sourceJob interface{}) {
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
	if jobMap, ok := sourceJob.(map[string]interface{}); ok {
		if steps, ok := jobMap["steps"].([]interface{}); ok {
			for _, step := range steps {
				if stepMap, ok := step.(map[string]interface{}); ok {
					if name, ok := stepMap["name"].(string); ok {
						if name == "Build" && hasBuildTarget {
							stepMap["run"] = "make build"
							fmt.Println("Replaced Build step with 'make build'")
						} else if name == "Test" && hasTestTarget {
							stepMap["run"] = "make test"
							fmt.Println("Replaced Test step with 'make test'")
						}
					}
				}
			}
		}
	}

	if !hasBuildTarget {
		fmt.Println("No 'build' target found in Makefile, Build step not replaced")
	}
	if !hasTestTarget {
		fmt.Println("No 'test' target found in Makefile, Test step not replaced")
	}
}

func commitAndPushWorkflow(destPath string) error {
	fmt.Println("Committing and pushing workflow changes...")

	// Get required environment variables
	githubToken := os.Getenv("GH_WORKFLOW_WRITE")
	githubRef := os.Getenv("GITHUB_REF")

	if githubToken == "" || githubRef == "" {
		return fmt.Errorf("required environment variables not set (GH_WORKFLOW_WRITE, GITHUB_REF)")
	}

	// Open the repository
	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get the worktree
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add the workflow file
	_, err = w.Add(destPath)
	if err != nil {
		return fmt.Errorf("failed to add workflow file: %w", err)
	}

	// Commit the changes
	commit, err := w.Commit("chore: update workflow", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "github-actions[bot]",
			Email: "github-actions[bot]@users.noreply.github.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	fmt.Printf("Committed changes: %s\n", commit.String())

	// Extract branch name from GITHUB_REF (refs/heads/main -> main)
	refSpec := strings.TrimPrefix(githubRef, "refs/heads/")
	if refSpec == "" {
		return fmt.Errorf("unsupported GITHUB_REF value %q: expected refs/heads/<branch>", githubRef)
	}
	branchRef := plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", refSpec))

	// Push the changes
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+%s:%s", branchRef, branchRef)),
		},
		Auth: &http.BasicAuth{
			Username: "x-access-token",
			Password: githubToken,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	fmt.Println("Successfully committed and pushed workflow changes")
	return nil
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

	// Define source and destination paths
	sourcePath := "templates/go.yml"
	destPath := ".github/workflows/main.yml"

	// Load source template
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source template: %v\n", err)
		os.Exit(1)
	}

	var sourceWorkflow OrderedMap
	if err := yaml.Unmarshal(sourceData, &sourceWorkflow); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing source template: %v\n", err)
		os.Exit(1)
	}

	// Extract the job from source workflow
	sourceJobs, ok := sourceWorkflow.Values["jobs"].(map[string]interface{})
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
		modifyJobForMakefile(sourceJob)
	}

	// Load or create destination workflow
	var destWorkflow OrderedMap
	if destData, err := os.ReadFile(destPath); err == nil {
		if err := yaml.Unmarshal(destData, &destWorkflow); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing destination workflow: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Create new workflow structure
		destWorkflow = OrderedMap{Keys: []string{}, Values: make(map[string]interface{})}
	}

	// Ensure jobs section exists
	if destWorkflow.Values["jobs"] == nil {
		destWorkflow.Values["jobs"] = make(map[string]interface{})
		if !slices.Contains(destWorkflow.Keys, "jobs") {
			destWorkflow.Keys = append(destWorkflow.Keys, "jobs")
		}
	}
	destJobs := destWorkflow.Values["jobs"].(map[string]interface{})

	// Add the job to destination workflow
	destJobs[sourceJobName] = sourceJob

	// Marshal back to YAML with 2-space indentation
	buf := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(destWorkflow); err != nil {
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
