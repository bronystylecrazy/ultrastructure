package main

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := di.App(
		di.Supply("Hellooo"),
		di.Invoke(func(l *zap.Logger, s string) {
			l.Info("a", zap.String("message", s))
		}, di.Name("prod")),
		di.Replace("Worldddd"),
		di.Invoke(func(l *zap.Logger, s string) {
			l.Info("b", zap.String("message", s))
		}, di.Name("dev")),
		di.Module("core",
			di.Provide(zap.NewProduction, di.Name("prod")),
			di.Provide(zap.NewExample, di.Name("dev")),
			// di.Replace(zap.NewDevelopment, di.Name("dev")),
			di.Invoke(func(l *zap.Logger) {
				l.Info("app-dev")
			}, di.Name("dev")),
			di.Invoke(func(l *zap.Logger) {
				l.Info("app-dev")
			}, di.Name("dev")),
			di.Replace(zap.NewNop, di.Name("dev")),
			di.Invoke(func(l *zap.Logger) {
				l.Info("app-prod")
			}, di.Name("prod")),
			di.Invoke(func(l *zap.Logger) {
				l.Info("app-prod")
			}, di.Name("prod")),
		),
	).Build()

	fx.New(fx.NopLogger, app).Run()
}
