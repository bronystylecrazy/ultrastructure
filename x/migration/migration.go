package migration

import (
	"embed"
	"errors"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNoSource = errors.New("migration: no source provided (did you forget to call migration.Use?)")
)

func Use(migrationsDirFS *embed.FS, paths ...string) di.Node {
	return di.Provide(func(gormCheck *database.GormChecker, config database.Config, db *gorm.DB, logger *zap.Logger) *Migration {
		return NewMigration(migrationsDirFS, config, db, logger, paths...)
	}, di.Params(``, ``, ``, di.Optional()))
}

func Run() di.Node {
	return di.Invoke(func(migration *Migration) error {
		if migration == nil {
			return ErrNoSource
		}
		return migration.Run()
	}, di.Params(di.Optional()))
}
