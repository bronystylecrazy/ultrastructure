package database

import "strings"

func ParseDialect(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "sqlite", "sqlite3":
		return "sqlite3"
	case "", "postgresql":
		return "postgres"
	default:
		return strings.ToLower(strings.TrimSpace(d))
	}
}
