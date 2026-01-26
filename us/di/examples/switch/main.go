package main

import (
	"os"

	"github.com/bronystylecrazy/ultrastructure/us/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	env := os.Getenv("APP_ENV")

	app := di.App(
		di.Switch(
			di.Case(env == "dev",
				di.Provide(zap.NewDevelopment),
			),
			di.WhenCase(func() bool {
				return env == "prod"
			}, di.Provide(zap.NewProduction)),
			di.DefaultCase(
				di.Provide(zap.NewExample),
			),
		),
		di.Invoke(func(l *zap.Logger) {
			l.Info("selected")
		}),
	).Build()

	fx.New(app).Run()
}
