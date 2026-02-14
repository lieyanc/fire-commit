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

// Tag creates a git tag with the given version string.
func Tag(version string) error {
	cmd := exec.Command("git", "tag", version)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git tag: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// PushTag pushes a specific tag to the remote.
func PushTag(tag string) error {
	cmd := exec.Command("git", "push", "origin", tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push tag: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// LatestTag returns the most recent tag reachable from HEAD.
// Returns an empty string if no tags exist.
func LatestTag() string {
	out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
