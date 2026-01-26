package main

import (
	"github.com/bronystylecrazy/ultrastructure/us/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		di.App(
			di.Decorate(func(l *zap.Logger) *zap.Logger {
				return l.With(zap.String("global", "true"))
			}, di.Name("prod")),
			di.Provide(
				zap.NewProduction,
				di.Name("prod"),
				di.Group("loggers"),
				di.Decorate(
					func(l *zap.Logger) *zap.Logger {
						return l.With(zap.String("ccccccc", "prod"))
					},
				),
			),
			di.Provide(
				zap.NewProduction,
				di.Name("dev"),
				di.Group("loggers"),
				di.Decorate(
					func(l *zap.Logger) *zap.Logger {
						return l.With(zap.String("ccccccc", "dev"))
					},
				),
			),
			di.Invoke(
				func(l *zap.Logger, r *zap.Logger) {
					l.With(zap.String("check", "yes")).Info("loggggg")
					r.With(zap.String("check", "yes")).Info("loggggg")
				},
				di.Name("prod"), di.Name("dev"),
			),
		).Build(),
	).Run()
}
