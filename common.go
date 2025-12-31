package main

import (
	"fmt"
	"os"
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

// addTrivySecuritySteps adds Trivy security scanning steps to the job
func addTrivySecuritySteps(job *Job) {
	if job == nil {
		return
	}

	// Always add secret scanning step
	secretScanStep := &Step{
		Name: "Scan for secrets with Trivy",
		Uses: "aquasecurity/trivy-action@0.33.1",
		With: map[string]interface{}{
			"scan-type": "fs",
			"scan-ref":  ".",
			"scanners":  "secret",
			"exit-code": "1",
			"severity":  "CRITICAL,HIGH",
			"format":    "sarif",
			"output":    "trivy-secrets.sarif",
		},
	}

	// Check if dependency scanning is applicable
	dependencyScanStep := createTrivyDependencyScanStep()

	// Insert security steps after checkout but before build steps
	var newSteps []*Step
	securityStepsInserted := false

	for _, step := range job.Steps {
		newSteps = append(newSteps, step)

		// Insert after checkout step
		if !securityStepsInserted && step.Uses != "" && strings.HasPrefix(step.Uses, "actions/checkout@") {
			if dependencyScanStep != nil {
				newSteps = append(newSteps, dependencyScanStep)
				fmt.Println("Added Trivy dependency scanning step")
			}
			newSteps = append(newSteps, secretScanStep)
			fmt.Println("Added Trivy secret scanning step")
			securityStepsInserted = true
		}
	}

	job.Steps = newSteps
}

// createTrivyDependencyScanStep creates a dependency scan step if supported files exist
func createTrivyDependencyScanStep() *Step {
	// Files that Trivy can scan for dependencies
	supportedFiles := []string{
		"package.json",       // npm
		"package-lock.json",  // npm
		"yarn.lock",          // yarn
		"pnpm-lock.yaml",     // pnpm
		"requirements.txt",   // pip
		"poetry.lock",        // poetry
		"Pipfile.lock",       // pipenv
		"go.mod",             // go
		"go.sum",             // go
		"pom.xml",            // maven
		"build.gradle",       // gradle
		"build.gradle.kts",   // gradle kotlin
		"*.csproj",           // nuget
		"packages.lock.json", // nuget
		"Gemfile.lock",       // bundler
		"Cargo.lock",         // rust
		"composer.lock",      // composer
		"mix.lock",           // hex
		"pubspec.lock",       // pub
	}

	// Check if any supported file exists
	for _, file := range supportedFiles {
		if strings.Contains(file, "*") {
			// Handle wildcard patterns
			entries, err := os.ReadDir(".")
			if err != nil {
				continue
			}
			pattern := strings.TrimPrefix(file, "*")
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), pattern) {
					fmt.Printf("Found dependency file: %s\n", entry.Name())
					return &Step{
						Name: "Scan dependencies with Trivy",
						Uses: "aquasecurity/trivy-action@0.33.1",
						With: map[string]interface{}{
							"scan-type": "fs",
							"scan-ref":  ".",
							"scanners":  "vuln",
							"exit-code": "0",
							"severity":  "CRITICAL,HIGH",
							"format":    "sarif",
							"output":    "trivy-results.sarif",
						},
					}
				}
			}
		} else {
			if _, err := os.Stat(file); err == nil {
				fmt.Printf("Found dependency file: %s\n", file)
				return &Step{
					Name: "Scan dependencies with Trivy",
					Uses: "aquasecurity/trivy-action@0.33.1",
					With: map[string]interface{}{
						"scan-type": "fs",
						"scan-ref":  ".",
						"scanners":  "vuln",
						"exit-code": "0",
						"severity":  "CRITICAL,HIGH",
						"format":    "sarif",
						"output":    "trivy-results.sarif",
					},
				}
			}
		}
	}

	fmt.Println("No supported dependency files found, skipping Trivy dependency scan")
	return nil
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

// applyActionInputs adds the with parameters to the pipeline-gen action step
func applyActionInputs(job *Job, packagesToInstall, fetchDepth string) {
	if packagesToInstall == "" && fetchDepth == "" {
		return
	}

	for _, step := range job.Steps {
		// Find the pipeline-gen action step
		if step.Uses != "" && strings.Contains(step.Uses, "automatic-pipeline-generator-action") {
			if step.With == nil {
				step.With = make(map[string]interface{})
			}

			// Only add parameters if they're not empty
			if packagesToInstall != "" {
				step.With["packages_to_install"] = packagesToInstall
				fmt.Printf("Adding packages_to_install to action: %s\n", packagesToInstall)
			}
			if fetchDepth != "" {
				step.With["fetch_depth"] = fetchDepth
				fmt.Printf("Adding fetch_depth to action: %s\n", fetchDepth)
			}
			break
		}
	}
}
