package ustest

import (
	"testing"

	us "github.com/bronystylecrazy/ultrastructure"
	"go.uber.org/fx/fxtest"
)

// App wraps fxtest.App with ultrastructure-specific constructor helpers.
type App struct {
	app *fxtest.App
}

// New builds a us.New app from nodes and returns a test app.
func New(t testing.TB, nodes ...any) *App {
	t.Helper()
	return &App{app: fxtest.New(t, us.New(nodes...).Build())}
}

// RequireStart starts the app and fails the test on error.
func (a *App) RequireStart() *App {
	a.app.RequireStart()
	return a
}

// RequireStop stops the app and fails the test on error.
func (a *App) RequireStop() {
	a.app.RequireStop()
}

// Fx exposes the underlying fxtest.App.
func (a *App) Fx() *fxtest.App {
	return a.app
}
