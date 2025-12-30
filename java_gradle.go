package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func detectJavaGradleProject() (bool, string) {
	indicators := []string{
		"gradlew",
		"build.gradle",
		"build.gradle.kts",
		"settings.gradle",
		"settings.gradle.kts",
	}

	for _, indicator := range indicators {
		if _, err := os.Stat(indicator); err == nil {
			return true, indicator
		}
	}

	return false, ""
}

func loadJavaGradleJobTemplate(packagesToInstall, fetchDepth, languageVersion string) (string, *Job, error) {
	jobTemplatePath := "templates/java_gradle.yml"

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
	applyJavaLanguageVersion(job, languageVersion)
	useGradleWrapper(job)
	addTrivySecuritySteps(job)

	return jobName, job, nil
}

func useGradleWrapper(job *Job) {
	if job == nil {
		return
	}

	if _, err := os.Stat("gradlew"); err != nil {
		return
	}

	fmt.Println("Found Gradle wrapper, switching to ./gradlew")

	for _, step := range job.Steps {
		if step.Run == "" {
			continue
		}
		step.Run = replaceCommandPrefix(step.Run, "gradle", "./gradlew")
	}

	wrapperStep := &Step{
		Name: "Ensure Gradle wrapper executable",
		Run:  "chmod +x gradlew",
	}

	insertStepAfterCheckout(job, wrapperStep)
}
