package main

import (
	"fmt"
	"strings"
)

// applyFetchDepth modifies the checkout step to include fetch-depth parameter
func applyFetchDepth(job *Job, fetchDepth string) {
	if fetchDepth == "" {
		return
	}

	fmt.Printf("Setting fetch depth to: %s\n", fetchDepth)

	for _, step := range job.Steps {
		// Find the checkout step
		if step.Uses != "" && strings.HasPrefix(step.Uses, "actions/checkout@") {
			if step.With == nil {
				step.With = make(map[string]interface{})
			}
			step.With["fetch-depth"] = fetchDepth
			break
		}
	}
}

// applyPackagesToInstall adds a step to install system packages
func applyPackagesToInstall(job *Job, packagesToInstall string) {
	if packagesToInstall == "" {
		return
	}

	fmt.Printf("Adding package installation step with packages: %s\n", packagesToInstall)

	// Parse the packages (space or comma separated)
	packages := strings.Fields(strings.ReplaceAll(packagesToInstall, ",", " "))
	if len(packages) == 0 {
		return
	}

	// Create the installation step
	installStep := &Step{
		Name: "Install build dependencies",
		Run:  generatePackageInstallScript(packages),
	}

	// Insert the install step after the checkout step
	var newSteps []*Step
	checkoutFound := false

	for _, step := range job.Steps {
		newSteps = append(newSteps, step)

		// Insert install step right after checkout
		if !checkoutFound && step.Uses != "" && strings.HasPrefix(step.Uses, "actions/checkout@") {
			newSteps = append(newSteps, installStep)
			checkoutFound = true
		}
	}

	// If no checkout step was found, prepend the install step
	if !checkoutFound {
		newSteps = append([]*Step{installStep}, newSteps...)
	}

	job.Steps = newSteps
}

// generatePackageInstallScript generates the apt-get install script for packages
func generatePackageInstallScript(packages []string) string {
	var script strings.Builder

	script.WriteString("sudo apt-get update\n")
	script.WriteString("sudo apt-get install -y")

	// Put all packages on the same line
	for _, pkg := range packages {
		script.WriteString(" ")
		script.WriteString(pkg)
	}

	return script.String()
}
