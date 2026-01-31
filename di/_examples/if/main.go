package main

import (
	"os"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	enableDev := os.Getenv("APP_ENV") != "prod"

	app := di.App(
		di.If(enableDev,
			di.Provide(zap.NewDevelopment),
		),
		di.When(func() bool { return os.Getenv("APP_ENV") == "prod" },
			di.Provide(zap.NewProduction),
		),
		di.Invoke(func(l *zap.Logger) {
			l.Info("ready")
		}),
	).Build()

	fx.New(app).Run()
}
