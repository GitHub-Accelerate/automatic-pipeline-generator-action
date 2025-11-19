package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

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
