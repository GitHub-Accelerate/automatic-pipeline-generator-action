package main

import (
	"fmt"
	"strings"
)

func insertStepAfterCheckout(job *Job, step *Step) {
	if job == nil || step == nil {
		return
	}

	inserted := false
	updated := make([]*Step, 0, len(job.Steps)+1)

	for _, existing := range job.Steps {
		updated = append(updated, existing)
		if !inserted && existing.Uses != "" && strings.HasPrefix(existing.Uses, "actions/checkout@") {
			updated = append(updated, step)
			inserted = true
		}
	}

	if inserted {
		job.Steps = updated
		return
	}

	job.Steps = append([]*Step{step}, job.Steps...)
}

func replaceCommandPrefix(script, command, replacement string) string {
	if script == "" || command == "" || replacement == "" {
		return script
	}

	lines := strings.Split(script, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !strings.HasPrefix(trimmed, command) {
			continue
		}

		if len(trimmed) > len(command) {
			next := trimmed[len(command)]
			if next != ' ' && next != '\t' && next != '\r' && next != '\n' && next != '-' {
				continue
			}
		}

		index := strings.Index(line, command)
		if index < 0 {
			continue
		}

		if index > 0 {
			prev := line[index-1]
			if prev == '.' || prev == '/' {
				continue
			}
		}

		lines[i] = line[:index] + replacement + line[index+len(command):]
	}

	return strings.Join(lines, "\n")
}

func applyJavaLanguageVersion(job *Job, languageVersion string) {
	if job == nil || languageVersion == "" {
		return
	}

	fmt.Printf("Setting Java language version to: %s\n", languageVersion)

	for _, step := range job.Steps {
		if step.Uses == "" || !strings.HasPrefix(step.Uses, "actions/setup-java@") {
			continue
		}

		if step.With == nil {
			step.With = make(map[string]interface{})
		}

		step.With["java-version"] = languageVersion
		return
	}
}
