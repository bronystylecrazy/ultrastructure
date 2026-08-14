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
	base := []any{otel.Providers(), database.Providers(), di.Supply(context.Background(), di.As[context.Context]())}
	return fx.New(di.App(append(base, nodes...)...).Build())
}

// Building GormOtel is what installs the gorm logger and the tracing plugin, so
// Use has to ask for it; nothing in an application does.
func TestUseBuildsGormOtel(t *testing.T) {
	var plugin *xgorm.GormOtel
	app := build(xgorm.Use(), di.Populate(&plugin))
	require.NoError(t, app.Err())
	assert.NotNil(t, plugin)
}

func TestUseProvidesGormHandle(t *testing.T) {
	var db *gorm.DB
	app := build(xgorm.Use(), di.Populate(&db))
	require.NoError(t, app.Err())
	assert.NotNil(t, db)
}

// The database here is unreachable. Without the option the application still
// builds and would only fail on its first query; with it the ping is reported
// while starting.
func TestCheckIsOptional(t *testing.T) {
	assert.NoError(t, build(xgorm.Use()).Err())
	assert.Error(t, build(xgorm.Use(xgorm.WithCheck())).Err())
}
