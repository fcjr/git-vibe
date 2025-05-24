package git

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repo struct {
	repo *git.Repository
}

func RepoFromWorkingDir() (*Repo, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}
	return &Repo{repo}, nil
}

// GetRecentCommits returns the recent commit messages.
// The limit parameter specifies the maximum number of recent commits to retrieve.
// The returned string contains the commit messages, separated by newlines.
func (r *Repo) GetRecentCommits(limit int) (string, error) {
	ref, err := r.repo.Head()
	if err != nil {
		return "", err
	}

	commitIter, err := r.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return "", err
	}
	defer commitIter.Close()

	var recentCommits string
	for i := 0; i < limit; i++ {
		commit, err := commitIter.Next()
		if err != nil {
			break
		}
		recentCommits += commit.Message
	}
	return recentCommits, nil
}

// Commit creates a new commit with the given message.
func (r *Repo) Commit(message string) error {
	w, err := r.repo.Worktree()
	if err != nil {
		return err
	}

	_, err = w.Commit(message, &git.CommitOptions{})
	if err != nil {
		return err
	}

	return nil
}

// GetStagedDiff returns the diff of the staged changes in the repository.
func (r *Repo) GetStagedDiff() (string, error) {
	w, err := r.repo.Worktree()
	if err != nil {
		return "", err
	}

	// Get repository status
	status, err := w.Status()
	if err != nil {
		return "", err
	}

	// Get HEAD commit for comparison
	head, err := r.repo.Head()
	if err != nil {
		return "", err
	}

	headCommit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return "", err
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", err
	}

	var diffOutput strings.Builder

	// Process each staged file
	for file, fileStatus := range status {
		if fileStatus.Staging == git.Unmodified {
			continue // Skip unstaged files
		}

		diffOutput.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))

		switch fileStatus.Staging {
		case git.Added:
			err := r.appendNewFileDiff(&diffOutput, w, file)
			if err != nil {
				diffOutput.WriteString(fmt.Sprintf("error reading new file %s: %v\n", file, err))
			}
		case git.Modified:
			err := r.appendModifiedFileDiff(&diffOutput, headTree, w, file)
			if err != nil {
				diffOutput.WriteString(fmt.Sprintf("error generating diff for %s: %v\n", file, err))
			}
		case git.Deleted:
			err := r.appendDeletedFileDiff(&diffOutput, headTree, file)
			if err != nil {
				diffOutput.WriteString(fmt.Sprintf("error reading deleted file %s: %v\n", file, err))
			}
		}
		diffOutput.WriteString("\n")
	}

	return diffOutput.String(), nil
}

func (r *Repo) appendNewFileDiff(output *strings.Builder, worktree *git.Worktree, filename string) error {
	file, err := worktree.Filesystem.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	output.WriteString("new file mode 100644\n")
	output.WriteString("index 0000000..0000000\n")
	output.WriteString(fmt.Sprintf("--- /dev/null\n"))
	output.WriteString(fmt.Sprintf("+++ b/%s\n", filename))

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line != "" || len(lines) > 1 {
			output.WriteString(fmt.Sprintf("+%s\n", line))
		}
	}

	return nil
}

func (r *Repo) appendModifiedFileDiff(output *strings.Builder, headTree *object.Tree, worktree *git.Worktree, filename string) error {
	// Get file from HEAD
	headFile, err := headTree.File(filename)
	if err != nil {
		return err
	}

	headContent, err := headFile.Contents()
	if err != nil {
		return err
	}

	// Get current staged content
	file, err := worktree.Filesystem.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	currentContentBytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	currentContent := string(currentContentBytes)

	// Generate diff header
	output.WriteString(fmt.Sprintf("index %s..%s 100644\n", "0000000", "0000000"))
	output.WriteString(fmt.Sprintf("--- a/%s\n", filename))
	output.WriteString(fmt.Sprintf("+++ b/%s\n", filename))

	// Simple line-by-line diff
	headLines := strings.Split(headContent, "\n")
	currentLines := strings.Split(currentContent, "\n")

	// Use a basic LCS-style diff algorithm
	r.generateUnifiedDiff(output, headLines, currentLines)

	return nil
}

// generateUnifiedDiff creates a unified diff between two sets of lines
func (r *Repo) generateUnifiedDiff(output *strings.Builder, oldLines, newLines []string) {
	i, j := 0, 0

	for i < len(oldLines) || j < len(newLines) {
		if i < len(oldLines) && j < len(newLines) {
			if oldLines[i] == newLines[j] {
				// Lines match
				output.WriteString(fmt.Sprintf(" %s\n", oldLines[i]))
				i++
				j++
			} else {
				// Lines differ - look ahead to find if this is a replacement or insertion/deletion
				foundMatch := false

				// Look ahead in new lines to see if old line appears later
				for k := j + 1; k < len(newLines) && k < j+5; k++ {
					if i < len(oldLines) && oldLines[i] == newLines[k] {
						// Found old line later in new - these are insertions
						for l := j; l < k; l++ {
							output.WriteString(fmt.Sprintf("+%s\n", newLines[l]))
						}
						output.WriteString(fmt.Sprintf(" %s\n", oldLines[i]))
						i++
						j = k + 1
						foundMatch = true
						break
					}
				}

				if !foundMatch {
					// Look ahead in old lines to see if new line appears later
					for k := i + 1; k < len(oldLines) && k < i+5; k++ {
						if j < len(newLines) && oldLines[k] == newLines[j] {
							// Found new line later in old - these are deletions
							for l := i; l < k; l++ {
								output.WriteString(fmt.Sprintf("-%s\n", oldLines[l]))
							}
							output.WriteString(fmt.Sprintf(" %s\n", newLines[j]))
							i = k + 1
							j++
							foundMatch = true
							break
						}
					}
				}

				if !foundMatch {
					// Simple replacement
					output.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
					output.WriteString(fmt.Sprintf("+%s\n", newLines[j]))
					i++
					j++
				}
			}
		} else if i < len(oldLines) {
			// Only old lines remaining (deletions)
			output.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
			i++
		} else if j < len(newLines) {
			// Only new lines remaining (additions)
			output.WriteString(fmt.Sprintf("+%s\n", newLines[j]))
			j++
		}
	}
}

func (r *Repo) appendDeletedFileDiff(output *strings.Builder, headTree *object.Tree, filename string) error {
	// Get file content from HEAD
	headFile, err := headTree.File(filename)
	if err != nil {
		return err
	}

	headContent, err := headFile.Contents()
	if err != nil {
		return err
	}

	output.WriteString("deleted file mode 100644\n")
	output.WriteString("index 0000000..0000000\n")
	output.WriteString(fmt.Sprintf("--- a/%s\n", filename))
	output.WriteString("+++ /dev/null\n")

	lines := strings.Split(headContent, "\n")
	for _, line := range lines {
		if line != "" || len(lines) > 1 {
			output.WriteString(fmt.Sprintf("-%s\n", line))
		}
	}

	return nil
}
