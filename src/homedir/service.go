package homedir

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bronystylecrazy/flexinfra/src/build"
	"github.com/bronystylecrazy/gx"
)

// service implements HomeDirService
type service struct {
	projectDir string
	isDev      bool
}

// NewService creates a new home directory service
// In development mode, uses repo root/.config
// In production mode, uses ~/.connectedtech/video-analytics-platform
func NewService() (Service, error) {
	isDev := !gx.IsTestEnv() && build.IsDevelopmentMode()

	var projectDir string
	if isDev {
		// Use repo root directory in development mode
		repoRoot, err := build.FindRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to find repo root: %w", err)
		}
		projectDir = filepath.Join(repoRoot, build.DevDirName)
	} else {
		// Use home directory in production mode
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		projectDir = filepath.Join(homeDir, build.ProjectDirName)
	}

	return &service{
		projectDir: projectDir,
		isDev:      isDev,
	}, nil
}

// GetProjectDir returns the full path to the project directory
func (s *service) GetProjectDir() (string, error) {
	return s.projectDir, nil
}

// EnsureProjectDir creates the project directory if it doesn't exist
func (s *service) EnsureProjectDir() error {
	if err := os.MkdirAll(s.projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	return nil
}
