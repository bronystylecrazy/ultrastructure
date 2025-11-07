package build

import (
	"os"
)

// IsDevelopmentMode checks if the application is running in development mode
// It checks for:
// 1. -debug flag in command line arguments
// 2. DEV environment variables
// 3. Running from a git repository (has .git directory)
func IsDevelopmentMode() bool {

	if Env == "development" {
		return true
	}

	// Check for -development flag in command line arguments
	for _, arg := range os.Args {
		if arg == "-development" || arg == "--development" {
			return true
		}
	}

	// Check DEV environment variable
	if os.Getenv("DEV") == "true" || os.Getenv("ENV") == "dev" || os.Getenv("ENVIRONMENT") == "development" {
		return true
	}

	// Check if running from a git repository (has .git directory)
	_, err := FindRepoRoot()
	return err == nil
}
