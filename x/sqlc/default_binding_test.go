package sqlc_test

import (
	"database/sql"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	"github.com/stretchr/testify/assert"
)

// An application that brings no database of its own takes sqlc's handles, and
// consumes them from the outer scope the way x/goose does.
func TestOuterScopeResolvesSqlcHandles(t *testing.T) {
	var db *sql.DB
	ustest.New(t,
		di.Replace(database.Config{
			Driver:     "postgres",
			Datasource: "postgres://postgres:postgres@localhost:5432/postgres",
		}),
		di.Populate(&db),
	).RequireStart().RequireStop()

	assert.NotNil(t, db)
}

// An application that provides its own handle outranks sqlc's default.
func TestApplicationHandleOutranksSqlcDefault(t *testing.T) {
	own := &sql.DB{}
	var got *sql.DB
	ustest.New(t,
		di.Replace(database.Config{
			Driver:     "postgres",
			Datasource: "postgres://postgres:postgres@localhost:5432/postgres",
		}),
		di.Provide(func() *sql.DB { return own }),
		di.Populate(&got),
	).RequireStart().RequireStop()

	assert.Same(t, own, got)
}
