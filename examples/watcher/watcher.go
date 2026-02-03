package main

import (
	"context"

	"go.uber.org/fx"
)

type AppWatcher struct {
	app *fx.App
}

func NewAppWatcher(app *fx.App) *AppWatcher {
	return &AppWatcher{app}
}

func (w *AppWatcher) Start() {
	w.app.Start(context.Background())
}

func (w *AppWatcher) Stop() {
	w.app.Stop(context.Background())
}

func main() {
	fx.New(
		fx.NopLogger,
		fx.Supply(fx.New()),
		fx.Provide(NewAppWatcher),
		fx.Invoke(func(w *AppWatcher) {
			w.Start()
		}),
	).Run()
}
