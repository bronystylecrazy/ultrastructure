package main

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := di.App(
		di.Decorate(func(l *zap.Logger) *zap.Logger {
			return l.With(zap.String("global", "true"))
		}),
		di.Decorate(func(l *zap.Logger) *zap.Logger {
			return l.With(zap.String("module", "core"))
		}),
		di.Module("core",
			di.Provide(zap.NewProduction, di.Name("prod")),
			di.Provide(zap.NewProduction, di.Group("loggers")),
			di.Invoke(func(l *zap.Logger) {
				l.Info("local")
			}, di.Name("prod")),
		),
		di.Decorate(func(loggers []*zap.Logger) []*zap.Logger {
			for i := range loggers {
				loggers[i] = loggers[i].With(
					zap.String("group", "loggers"),
					zap.String("global", "true"),
					zap.String("module", "core"),
				)
			}
			return loggers
		}, di.Group("loggers")),
		di.Invoke(func(loggers []*zap.Logger) {
			for _, lg := range loggers {
				lg.Info("grouped")
			}
		}, di.Group("loggers")),
		di.Invoke(func(l *zap.Logger) {
			l.Info("global")
		}, di.Name("prod")),
	).Build()

	fx.New(app).Run()
}
