package build

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRepoRoot finds the root directory of the git repository
func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree until we find .git
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root without finding .git
			return "", fmt.Errorf("not in a git repository")
		}
		dir = parent
	}
}
