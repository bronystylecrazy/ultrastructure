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

func UseMigrations(migrationsDirFS *embed.FS, paths ...string) di.Node {
	return di.Invoke(func(config Config, db *gorm.DB, log *zap.Logger) error {
		if !config.Migrate || migrationsDirFS == nil {
			log.Info("Skipping migrations")
			return nil
		}

		base := fs.FS(migrationsDirFS)
		path := defaultMigrationPath
		if len(paths) > 0 && paths[0] != "" {
			path = paths[0]
		}

		sub, err := fs.Sub(migrationsDirFS, path)
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
	})
}
