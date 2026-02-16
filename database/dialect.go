package database

import (
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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

func NewDialector(config Config) gorm.Dialector {
	switch strings.ToLower(strings.TrimSpace(config.Dialect)) {
	case "", "postgres", "postgresql":
		return postgres.Open(config.Datasource)
	case "mysql":
		return mysql.Open(config.Datasource)
	case "sqlite", "sqlite3":
		return sqlite.Open(config.Datasource)
	default:
		// Keep backward-compatible behavior: fallback to postgres.
		return postgres.Open(config.Datasource)
	}
}

// NewPostgresDialector is kept for backward compatibility.
func NewPostgresDialector(config Config) gorm.Dialector {
	return NewDialector(config)
}
