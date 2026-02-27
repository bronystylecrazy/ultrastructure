package sqlc_test

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	"github.com/bronystylecrazy/ultrastructure/x/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestSqlcRunner(t *testing.T) {
	var pool *pgxpool.Pool
	ustest.New(t,
		di.Replace(database.Config{
			Driver:     "postgres",
			Datasource: "postgres://postgres:postgres@localhost:5432/postgres",
		}),
		di.Populate(&pool),
	).RequireStart().RequireStop()

	assert.NotNil(t, pool)
}

type testQueryFactory struct {
	Pool *pgxpool.Pool
}

type testDBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type testQueries struct {
	db testDBTX
}

type testQuerier interface {
	DB() testDBTX
}

type testQuerierImpl struct {
	db testDBTX
}

func (q *testQuerierImpl) DB() testDBTX {
	return q.db
}

func TestProvideCallsCtorWithPool(t *testing.T) {
	var got testQueryFactory
	pool := &pgxpool.Pool{}

	ustest.New(t,
		di.Replace(pool),
		sqlc.Provide(func(db *pgxpool.Pool) testQueryFactory {
			return testQueryFactory{Pool: db}
		}),
		di.Populate(&got),
	).RequireStart().RequireStop()

	assert.Same(t, pool, got.Pool)
}

func TestProvideCallsCtorWithDBTX(t *testing.T) {
	var got *testQueries
	pool := &pgxpool.Pool{}

	ustest.New(t,
		di.Replace(pool),
		sqlc.Provide(func(db testDBTX) *testQueries {
			return &testQueries{db: db}
		}),
		di.Populate(&got),
	).RequireStart().RequireStop()

	assert.NotNil(t, got)
	assert.Same(t, pool, got.db)
}

func TestProvidePassesOptionsToInnerProvide(t *testing.T) {
	var got testQuerier
	pool := &pgxpool.Pool{}

	ustest.New(t,
		di.Replace(pool),
		sqlc.Provide(func(db testDBTX) *testQuerierImpl {
			return &testQuerierImpl{db: db}
		}, di.As[testQuerier]()),
		di.Populate(&got),
	).RequireStart().RequireStop()

	assert.NotNil(t, got)
	assert.Same(t, pool, got.DB())
}
