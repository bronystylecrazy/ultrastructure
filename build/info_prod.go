//go:build prod

package build

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var Name = "ultrastructure"
var Version = "v0.0.0-production"
var BuildDate = time.Now().Format("2006-01-02 15:04:05")
var Commit = "unknown"
var Mode = ModeProduction
var EnableDebug = false
var ProjectDir string = ".connectedtech/ultrastructure"
var Env = "production"

func GetProjectDir() (string, error) {
	// Use home directory in production mode
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	projectDir := filepath.Join(homeDir, ProjectDir)
	return projectDir, nil
}
