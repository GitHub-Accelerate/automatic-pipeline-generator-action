package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func detectJavaMavenProject() (bool, string) {
	if _, err := os.Stat("pom.xml"); err == nil {
		return true, "pom.xml"
	}
	return false, ""
}

func loadJavaMavenJobTemplate(packagesToInstall, fetchDepth, languageVersion string) (string, *Job, error) {
	jobTemplatePath := "templates/java_maven.yml"

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
	useMavenWrapper(job)

	return jobName, job, nil
}

func useMavenWrapper(job *Job) {
	if job == nil {
		return
	}

	if _, err := os.Stat("mvnw"); err != nil {
		return
	}

	fmt.Println("Found Maven wrapper, switching to ./mvnw")

	for _, step := range job.Steps {
		if step.Run == "" {
			continue
		}
		step.Run = replaceCommandPrefix(step.Run, "mvn", "./mvnw")
	}

	wrapperStep := &Step{
		Name: "Ensure Maven wrapper executable",
		Run:  "chmod +x mvnw",
	}

	insertStepAfterCheckout(job, wrapperStep)
}
