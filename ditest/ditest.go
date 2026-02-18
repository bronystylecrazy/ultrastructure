package ditest

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx/fxtest"
)

// App wraps fxtest.App with DI-specific constructor helpers.
type App struct {
	app *fxtest.App
}

// New builds a di.App from nodes and returns a test app.
func New(t testing.TB, nodes ...any) *App {
	t.Helper()
	return &App{app: fxtest.New(t, di.App(nodes...).Build())}
}

// RequireStart starts the app and fails the test on error.
func (a *App) RequireStart() *App {
	a.app.RequireStart()
	return a
}

// RequireStop stops the app and fails the test on error.
func (a *App) RequireStop() *App {
	a.app.RequireStop()
	return a
}

// RequiredStop is an alias of RequireStop for convenience.
func (a *App) RequiredStop() *App {
	return a.RequireStop()
}

// Fx exposes the underlying fxtest.App.
func (a *App) Fx() *fxtest.App {
	return a.app
}
