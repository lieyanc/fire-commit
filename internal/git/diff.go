package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// StagedDiff returns the diff of staged changes.
func StagedDiff(maxLines int) (string, error) {
	out, err := exec.Command("git", "diff", "--cached").Output()
	if err != nil {
		return "", fmt.Errorf("git diff --cached: %w", err)
	}
	return truncateDiff(string(out), maxLines), nil
}

// AllDiff returns the diff of all changes (staged + unstaged).
func AllDiff(maxLines int) (string, error) {
	out, err := exec.Command("git", "diff", "HEAD").Output()
	if err != nil {
		// HEAD might not exist (initial commit), fall back to diff of staged
		out2, err2 := exec.Command("git", "diff", "--cached").Output()
		if err2 != nil {
			return "", fmt.Errorf("git diff: %w", err)
		}
		return truncateDiff(string(out2), maxLines), nil
	}
	return truncateDiff(string(out), maxLines), nil
}

// StagedFileNames returns a list of staged file names.
func StagedFileNames() ([]string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached --name-only: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// StageAll runs git add -A to stage all changes.
func StageAll() error {
	return exec.Command("git", "add", "-A").Run()
}

// DiffStat returns a short stat summary of the staged diff.
func DiffStat() (string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--stat").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func truncateDiff(diff string, maxLines int) string {
	if maxLines <= 0 {
		return diff
	}
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}
	truncated := strings.Join(lines[:maxLines], "\n")
	truncated += fmt.Sprintf("\n\n... (truncated %d lines)", len(lines)-maxLines)
	return truncated
}
