package sqlc

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Providers() di.Node {
	return di.Options(
		di.Provide(NewPool),
		di.Provide(NewDB),
	)
}

func Provide[DB any, T any](ctor func(DB) T, opts ...any) di.Node {
	allOpts := append([]any{di.Params(di.Optional(), di.Optional())}, opts...)
	return di.Provide(func(pool *pgxpool.Pool, db *sql.DB) (T, error) {
		var zero T
		if pool != nil {
			if in, ok := any(pool).(DB); ok {
				return ctor(in), nil
			}
		}
		if db != nil {
			if in, ok := any(db).(DB); ok {
				return ctor(in), nil
			}
		}
		return zero, fmt.Errorf("sqlc: no compatible database handle for ctor argument")
	}, allOpts...)
}

func NewDB(lc fx.Lifecycle, config database.Config, pool *pgxpool.Pool, logger *zap.Logger) (*sql.DB, error) {
	log := logger.Named("sqlc")

	if pool != nil {
		log.Debug("creating database/sql connection from pgx pool")
		db := sql.OpenDB(stdlib.GetPoolConnector(pool))
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return db.Close()
			},
		})
		return db, nil
	}

	driverName, err := sqlDriverName(config.Driver)
	if err != nil {
		return nil, err
	}
	log.Debug("creating database/sql connection", zap.String("driver", driverName))
	db, err := sql.Open(driverName, config.Datasource)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})
	return db, nil
}

func sqlDriverName(driver string) (string, error) {
	switch database.ParseDialect(driver) {
	case "postgres":
		return "pgx", nil
	case "mysql":
		return "mysql", nil
	case "sqlite3":
		return "sqlite3", nil
	default:
		return "", fmt.Errorf("sqlc: unsupported database driver %q", driver)
	}
}
