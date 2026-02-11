package database

import (
	"embed"
	"io/fs"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var defaultMigrationPath = "migrations"

type migrationSource struct {
	FS    *embed.FS
	Paths []string
}

// UseMigrations registers migration files in the DI container.
func UseMigrations(migrationsDirFS *embed.FS, paths ...string) di.Node {
	return di.Supply(migrationSource{
		FS:    migrationsDirFS,
		Paths: paths,
	})
}

// RunMigrations runs goose migrations using the source registered by UseMigrations.
func RunMigrations() di.Node {
	return di.Invoke(runMigrations)
}

func runMigrations(config Config, db *gorm.DB, log *zap.Logger, source migrationSource) error {
	if !config.Migrate || source.FS == nil {
		log.Info("Skipping migrations")
		return nil
	}

	base := fs.FS(source.FS)
	path := defaultMigrationPath
	if len(source.Paths) > 0 && source.Paths[0] != "" {
		path = source.Paths[0]
	}

	sub, err := fs.Sub(source.FS, path)
	if err != nil {
		return err
	}

	base = sub
	path = "."
	goose.SetBaseFS(base)

	if err := goose.SetDialect(config.Dialect); err != nil {
		log.Error("Failed to set dialect", zap.Error(err))
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Error("Failed to get database connection", zap.Error(err))
		return err
	}

	goose.SetLogger(goose.NopLogger())

	return goose.Up(sqlDB, path)
}
