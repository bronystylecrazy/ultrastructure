//go:build windows

package service

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultWindowsProgramData = `C:\ProgramData`
const defaultWindowsLogFile = "service.log"

// WindowsServiceLogFile returns the daemon log file path for a Windows service.
func WindowsServiceLogFile(name string) string {
	serviceName := sanitizeWindowsPathSegment(name)
	if serviceName == "" {
		serviceName = "service"
	}

	base := strings.TrimSpace(os.Getenv("PROGRAMDATA"))
	if base == "" {
		base = defaultWindowsProgramData
	}
	return filepath.Join(base, serviceName, defaultWindowsLogFile)
}

func sanitizeWindowsPathSegment(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
