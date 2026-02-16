package database

import (
	"embed"
	"errors"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNoMigrationsSource = errors.New("database: no migration source provided (did you forget to call UseMigrations?)")
)

// UseMigrations registers migration files in the DI container.
func UseMigrations(migrationsDirFS *embed.FS, paths ...string) di.Node {
	return di.Provide(func(gormCheck *GormChecker, config Config, db *gorm.DB, logger *zap.Logger) *Migration {
		return NewMigration(migrationsDirFS, config, db, logger, paths...)
	}, di.Params(``, ``, ``, di.Optional()))
}

// RunMigrations runs goose migrations using the source registered by UseMigrations.
func RunMigrations() di.Node {
	return di.Invoke(func(migration *Migration) error {
		if migration == nil {
			return ErrNoMigrationsSource
		}
		return migration.Run()
	}, di.Params(di.Optional()))
}

func Module(opts ...di.Node) di.Node {
	return di.Module(
		"us/database",
		di.Config[Config]("db"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
		di.Provide(NewDialector),
		di.Provide(NewGormDB),
		di.Provide(NewGormChecker),
		di.Options(di.ConvertAnys(opts)...),
	)
}
