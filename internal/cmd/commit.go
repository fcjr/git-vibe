package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"text/template"

	"github.com/fcjr/git-vibe/internal/git"
	"github.com/fcjr/git-vibe/internal/validators"
	"github.com/ollama/ollama/api"
	"github.com/spf13/cobra"
)

type PromptData struct {
	StagedDiff    string
	RecentCommits string
}

var promptTemplate = `
# System Prompt

You are an expert software developer tasked with generating clear, concise, and meaningful commit messages. Your job is to analyze the staged changes (git diff) and recent commit history to create a commit message that follows best practices.
Based on the provided staged changes and recent commit history, generate a commit message that:

Follows conventional commit format when appropriate (feat, fix, docs, style, refactor, test, chore)
Uses imperative mood (e.g., "Add feature" not "Added feature")
Is concise but descriptive (50 characters or less for the subject line)
Includes a body if needed to explain the "why" behind complex changes
Maintains consistency with the project's existing commit style
Groups related changes logically

## Context

Staged Changes:

{{ .StagedDiff }}

## Instructions: Provide only the commit message in this format:

<type>(<scope>): <subject>

<optional body>

<optional footer>

## Guidelines:

Subject line: Start with lowercase unless it's a proper noun
Body: Wrap at 72 characters, explain what and why (not how)
Scope: Use if the project commonly uses scopes (api, ui, db, etc.)
Breaking changes: Include "BREAKING CHANGE:" in footer if applicable
Issue references: Include if following project conventions

Analysis steps:

Identify the primary change type from the diff
Look for patterns in recent commits to match style
Determine scope if the project uses them consistently
Consider impact - is this a feature, bugfix, or maintenance?
Check for breaking changes that need special notation

Generate a single, well-formatted commit message based on this analysis.
Reply ONLY with the commit message as this will be used programmatically.
DO NOT EXPLAIN YOUR WORK.
`

func generatePrompt(repo *git.Repo) (string, error) {

	stagedDiff, err := repo.GetStagedDiff()
	if err != nil {
		return "", err
	}

	recentCommits, err := repo.GetRecentCommits(10)
	if err != nil {
		return "", err
	}

	tmpl := template.Must(template.New("prompt").Parse(promptTemplate))

	templateData := &PromptData{
		RecentCommits: recentCommits,
		StagedDiff:    stagedDiff,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const model = "qwen2.5-coder:1.5b"

var commitCmd = &cobra.Command{
	Use:   "commit",
	Args:  validators.NoArgs(),
	Short: "Commits the currently staged files using an llm generated commit message",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background() // does cmd's ctx have a timeout set?

		client, err := api.ClientFromEnvironment()
		if err != nil {
			log.Fatal(err)
		}

		// pullReq := &api.PullRequest{
		// 	Model: model,
		// }
		// progressFunc := func(resp api.ProgressResponse) error {
		// 	fmt.Printf("Progress: status=%v, total=%v, completed=%v\n", resp.Status, resp.Total, resp.Completed)
		// 	return nil
		// }

		// err = client.Pull(ctx, pullReq, progressFunc)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		repo, err := git.RepoFromWorkingDir()
		if err != nil {
			log.Fatal(err)
		}

		prompt, err := generatePrompt(repo)
		if err != nil {
			log.Fatal(err)
		}

		req := &api.GenerateRequest{
			Model:  model,
			Prompt: prompt,

			// set streaming to false
			Stream: new(bool),
		}

		respChan := make(chan api.GenerateResponse, 1)
		respFunc := func(resp api.GenerateResponse) error {
			respChan <- resp
			return nil
		}
		err = client.Generate(ctx, req, respFunc)
		if err != nil {
			log.Fatal(err)
		}
		resp := <-respChan

		err = repo.Commit(resp.Response)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Commited with message: ", resp.Response)
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
}
