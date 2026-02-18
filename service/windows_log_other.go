//go:build !windows

package service

// WindowsServiceLogFile is a no-op placeholder on non-Windows platforms.
func WindowsServiceLogFile(name string) string {
	return ""
}
