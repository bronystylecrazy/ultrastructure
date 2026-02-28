package goose

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/zap"
)

var (
	ErrNoSource = errors.New("goose: no source provided (did you forget to call goose.Use?)")
)

func Use(migrationsDirFS *embed.FS, opts ...Option) di.Node {
	cfg := parseOptions(opts...)
	migration := migrationProvider(migrationsDirFS, cfg)

	return di.Provide(
		migration,
		di.Params(``, ``, di.Optional()),
	)
}

func Run() di.Node {
	return di.Invoke(func(m *Goose) error {
		if m == nil {
			return ErrNoSource
		}
		return m.Run()
	}, di.Params(di.Optional()))
}

func UseMigrationCommands() di.Node {
	return di.Options(
		di.Provide(NewGooseCommand),
		di.Provide(NewMigrateCommand),
	)
}

func parseOptions(opts ...Option) option {
	cfg := option{}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&cfg)
		}
	}
	return cfg
}

func migrationProvider(migrationsDirFS *embed.FS, cfg option) func(config database.Config, db *sql.DB, logger *zap.Logger) *Goose {
	return func(config database.Config, db *sql.DB, logger *zap.Logger) *Goose {
		migrationLogger := cfg.logger
		if migrationLogger == nil {
			migrationLogger = logger
		}
		return NewGoose(migrationsDirFS, config, db, migrationLogger, cfg.paths...)
	}
}
