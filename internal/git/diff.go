package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kujtimiihoxha/termai/internal/config"
)

type DiffStats struct {
	Additions int
	Removals  int
}

func GenerateGitDiff(filePath string, contentBefore string, contentAfter string) (string, error) {
	tempDir, err := os.MkdirTemp("", "git-diff-temp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		return "", fmt.Errorf("failed to initialize git repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	fullPath := filepath.Join(tempDir, filePath)
	if err = os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create directories: %w", err)
	}
	if err = os.WriteFile(fullPath, []byte(contentBefore), 0o644); err != nil {
		return "", fmt.Errorf("failed to write 'before' content: %w", err)
	}

	_, err = wt.Add(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to add file to git: %w", err)
	}

	beforeCommit, err := wt.Commit("Before", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OpenCode",
			Email: "coder@opencode.ai",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit 'before' version: %w", err)
	}

	if err = os.WriteFile(fullPath, []byte(contentAfter), 0o644); err != nil {
		return "", fmt.Errorf("failed to write 'after' content: %w", err)
	}

	_, err = wt.Add(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to add updated file to git: %w", err)
	}

	afterCommit, err := wt.Commit("After", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OpenCode",
			Email: "coder@opencode.ai",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit 'after' version: %w", err)
	}

	beforeCommitObj, err := repo.CommitObject(beforeCommit)
	if err != nil {
		return "", fmt.Errorf("failed to get 'before' commit: %w", err)
	}

	afterCommitObj, err := repo.CommitObject(afterCommit)
	if err != nil {
		return "", fmt.Errorf("failed to get 'after' commit: %w", err)
	}

	patch, err := beforeCommitObj.Patch(afterCommitObj)
	if err != nil {
		return "", fmt.Errorf("failed to generate patch: %w", err)
	}

	return patch.String(), nil
}

func GenerateGitDiffWithStats(filePath string, contentBefore string, contentAfter string) (string, DiffStats, error) {
	tempDir, err := os.MkdirTemp("", "git-diff-temp")
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to initialize git repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to get worktree: %w", err)
	}

	fullPath := filepath.Join(tempDir, filePath)
	if err = os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to create directories: %w", err)
	}
	if err = os.WriteFile(fullPath, []byte(contentBefore), 0o644); err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to write 'before' content: %w", err)
	}

	_, err = wt.Add(filePath)
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to add file to git: %w", err)
	}

	beforeCommit, err := wt.Commit("Before", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OpenCode",
			Email: "coder@opencode.ai",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to commit 'before' version: %w", err)
	}

	if err = os.WriteFile(fullPath, []byte(contentAfter), 0o644); err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to write 'after' content: %w", err)
	}

	_, err = wt.Add(filePath)
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to add updated file to git: %w", err)
	}

	afterCommit, err := wt.Commit("After", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "OpenCode",
			Email: "coder@opencode.ai",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to commit 'after' version: %w", err)
	}

	beforeCommitObj, err := repo.CommitObject(beforeCommit)
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to get 'before' commit: %w", err)
	}

	afterCommitObj, err := repo.CommitObject(afterCommit)
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to get 'after' commit: %w", err)
	}

	patch, err := beforeCommitObj.Patch(afterCommitObj)
	if err != nil {
		return "", DiffStats{}, fmt.Errorf("failed to generate patch: %w", err)
	}

	stats := DiffStats{}
	for _, fileStat := range patch.Stats() {
		stats.Additions += fileStat.Addition
		stats.Removals += fileStat.Deletion
	}

	return patch.String(), stats, nil
}

func FormatDiff(diffText string, width int) (string, error) {
	if isSplitDiffsAvailable() {
		return formatWithSplitDiffs(diffText, width)
	}

	return formatSimple(diffText), nil
}

func isSplitDiffsAvailable() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

func formatWithSplitDiffs(diffText string, width int) (string, error) {
	var cmd *exec.Cmd

	appCfg := config.Get()
	appWd := config.WorkingDirectory()
	script := filepath.Join(
		appWd,
		appCfg.Data.Directory,
		"diff",
		"index.mjs",
	)

	cmd = exec.Command("node", script, "--color")

	cmd.Env = append(os.Environ(), fmt.Sprintf("COLUMNS=%d", width))

	cmd.Stdin = strings.NewReader(diffText)

	var out bytes.Buffer
	cmd.Stdout = &out

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git-split-diffs error: %v, stderr: %s", err, stderr.String())
	}

	return out.String(), nil
}

func formatSimple(diffText string) string {
	lines := strings.Split(diffText, "\n")
	var result strings.Builder

	for _, line := range lines {
		if len(line) == 0 {
			result.WriteString("\n")
			continue
		}

		switch line[0] {
		case '+':
			result.WriteString("\033[32m" + line + "\033[0m\n")
		case '-':
			result.WriteString("\033[31m" + line + "\033[0m\n")
		case '@':
			result.WriteString("\033[36m" + line + "\033[0m\n")
		case 'd':
			if strings.HasPrefix(line, "diff --git") {
				result.WriteString("\033[1m" + line + "\033[0m\n")
			} else {
				result.WriteString(line + "\n")
			}
		default:
			result.WriteString(line + "\n")
		}
	}

	if !strings.HasSuffix(diffText, "\n") {
		output := result.String()
		return output[:len(output)-1]
	}

	return result.String()
}
