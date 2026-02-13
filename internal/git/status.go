package git

import (
	"os/exec"
	"strings"
)

// HasStagedChanges returns true if there are staged changes in the index.
func HasStagedChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means there are differences
			if exitErr.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, err
	}
	return false, nil
}

// HasUnstagedChanges returns true if there are unstaged changes in the working tree.
func HasUnstagedChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--quiet")
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, err
	}
	return false, nil
}

// HasUntrackedFiles returns true if there are untracked files.
func HasUntrackedFiles() (bool, error) {
	out, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// IsGitRepo returns true if the current directory is inside a git repository.
func IsGitRepo() bool {
	err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run()
	return err == nil
}
