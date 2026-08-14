package xgorm_test

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	xgorm "github.com/bronystylecrazy/ultrastructure/x/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func build(nodes ...any) *fx.App {
	base := []any{
		otel.Providers(),
		database.Providers(),
		di.Supply(context.Background(), di.As[context.Context]()),
	}
	return fx.New(di.App(append(base, nodes...)...).Build())
}

func reachable() di.Node {
	return di.Replace(database.Config{Driver: "sqlite", Datasource: "file::memory:?cache=shared"})
}

// Unreachable: an empty datasource falls back to postgres and dials a socket
// that is not there.
func unreachable() di.Node {
	return di.Replace(database.Config{Driver: "postgres"})
}

// Building GormOtel is what installs the gorm logger and the tracing plugin, so
// Use has to ask for it; nothing in an application does.
func TestUseBuildsGormOtel(t *testing.T) {
	var plugin *xgorm.GormOtel
	app := build(reachable(), xgorm.Use(), di.Populate(&plugin))
	require.NoError(t, app.Err())
	assert.NotNil(t, plugin)
}

func TestUseProvidesGormHandle(t *testing.T) {
	var db *gorm.DB
	app := build(reachable(), xgorm.Use(), di.Populate(&db))
	require.NoError(t, app.Err())
	assert.NotNil(t, db)
}

func TestCheckPassesAgainstAReachableDatabase(t *testing.T) {
	assert.NoError(t, build(reachable(), xgorm.Use(xgorm.WithCheck())).Err())
}

// Without the option the application still builds and would only fail on its
// first query; with it the ping is reported while starting.
func TestCheckIsOptional(t *testing.T) {
	assert.NoError(t, build(unreachable(), xgorm.Use()).Err())
	assert.Error(t, build(unreachable(), xgorm.Use(xgorm.WithCheck())).Err())
}
