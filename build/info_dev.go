//go:build !prod

package build

import (
	"fmt"
	"path/filepath"
	"time"
)

var Name = "ultrastructure"
var Version = "v0.0.0-development"
var BuildDate = time.Now().Format("2006-01-02 15:04:05")
var Commit = "unknown"
var Mode = ModeDevelopment
var EnableDebug = true
var ProjectDir string = "."
var Env = "development"

// GetProjectDir returns the full path to the project directory
func GetProjectDir() (string, error) {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find repo root: %w", err)
	}
	return filepath.Join(repoRoot, ProjectDir), nil
}
