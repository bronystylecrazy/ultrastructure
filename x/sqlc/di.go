package sqlc

import (
	"database/sql"
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func Providers() di.Node {
	return di.Options(
		di.Provide(NewPool),
		di.Provide(func(pool *pgxpool.Pool, db *sql.DB, logger *zap.Logger) *sql.DB {
			if db != nil {
				logger.Named("sqlc").Debug("using provided database connection")
				return db
			}
			logger.Named("sqlc").Debug("creating database connection from pool")
			return sql.OpenDB(stdlib.GetPoolConnector(pool))
		}, di.Params(nil, di.Optional(), nil)),
	)
}

func Provide[DB any, T any](ctor func(DB) T, opts ...any) di.Node {
	return di.Provide(func(pool *pgxpool.Pool) (T, error) {
		var zero T
		db, ok := any(pool).(DB)
		if !ok {
			return zero, fmt.Errorf("sqlc: pool (%T) is not assignable to ctor arg", pool)
		}
		return ctor(db), nil
	}, opts...)
}
