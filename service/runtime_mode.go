package service

// Mode describes how the current process is running.
type Mode string

const (
	ModeCLI    Mode = "cli"
	ModeDaemon Mode = "daemon"
)
