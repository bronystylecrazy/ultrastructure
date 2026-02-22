package xgorm

import (
	"github.com/bronystylecrazy/ultrastructure/database"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDialector(config database.Config) gorm.Dialector {
	switch database.ParseDialect(config.Driver) {
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
func NewPostgresDialector(config database.Config) gorm.Dialector {
	return NewDialector(config)
}
