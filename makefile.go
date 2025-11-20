package main

import (
	"fmt"
	"os"
	"strings"
)

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
	buildReplaced := false
	testReplaced := false
	for _, step := range job.Steps {
		stepNameLower := strings.ToLower(step.Name)
		// Exact match for "build" or "test" step names
		if !buildReplaced && hasBuildTarget && stepNameLower == "build" {
			step.Run = "make -j$(nproc) build"
			fmt.Printf("Replaced '%s' step with 'make build'\n", step.Name)
			buildReplaced = true
		} else if !testReplaced && hasTestTarget && stepNameLower == "test" {
			step.Run = "make test"
			fmt.Printf("Replaced '%s' step with 'make test'\n", step.Name)
			testReplaced = true
		}
	}

	if !hasBuildTarget {
		fmt.Println("No 'build' target found in Makefile, Build step not replaced")
	}
	if !hasTestTarget {
		fmt.Println("No 'test' target found in Makefile, Test step not replaced")
	}
}
