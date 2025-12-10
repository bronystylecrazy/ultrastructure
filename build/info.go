package build

import (
	"fmt"
	"os"
	"path/filepath"
)

func IsDevelopment() bool {
	return Mode == ModeDevelopment
}

func IsProduction() bool {
	return Mode == ModeProduction
}

// EnsureProjectDir creates the project directory if it doesn't exist
func EnsureProjectDir() error {
	projectDir, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}
	return os.MkdirAll(projectDir, 0755)
}

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
