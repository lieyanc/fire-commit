package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Commit creates a git commit with the given message.
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// Push pushes the current branch to its upstream remote.
func Push() error {
	cmd := exec.Command("git", "push")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// CurrentBranch returns the name of the current branch.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
