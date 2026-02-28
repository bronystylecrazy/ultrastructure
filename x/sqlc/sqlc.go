package sqlc

import (
	"context"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

func NewPool(lc fx.Lifecycle, config database.Config) (*pgxpool.Pool, error) {
	if database.ParseDialect(config.Driver) != "postgres" {
		return nil, nil
	}

	pool, err := pgxpool.New(context.Background(), config.Datasource)

	if err != nil {
		return nil, err
	}

	lc.Append(
		fx.Hook{
			OnStop: func(ctx context.Context) error {
				pool.Close()
				return nil
			},
		},
	)

	return pool, nil
}
