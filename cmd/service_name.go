package cmd

import "strings"

func sanitizeServiceName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		return "ultrastructure"
	}
	base = strings.ReplaceAll(base, " ", "-")
	return base
}
