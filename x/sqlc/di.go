package sqlc

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Providers() di.Node {
	return di.Options(
		di.Provide(NewPool),
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
