package homedir

// HomeDirService defines the interface for managing project home directory
type Service interface {
	// GetProjectDir returns the full path to the project directory
	GetProjectDir() (string, error)
	// EnsureProjectDir creates the project directory if it doesn't exist
	EnsureProjectDir() error
}
