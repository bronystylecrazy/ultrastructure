package web

import "go.uber.org/fx"

func registerLifecycle(lc fx.Lifecycle, app App) {
	lc.Append(fx.Hook{
		OnStart: app.Start,
		OnStop:  app.Stop,
	})
}
