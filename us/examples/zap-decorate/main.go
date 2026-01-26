package main

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/bronystylecrazy/ultrastructure/us"
)

func main() {
	app := fx.New(
		us.Module(
			"core",
			us.Provide(func() (*zap.Logger, error) {
				return zap.NewProduction()
			}),
		),
		us.Module(
			"logging",
			us.Decorate(func(log *zap.Logger) *zap.Logger {
				return log.With(zap.String("module", "logging"))
			}),
			us.Invoke(func(log *zap.Logger) {
				log.Info("hello from logging module")
			}),
		),
		us.Invoke(func(log *zap.Logger) {
			log.With(zap.String("from", "invoke")).Info("hello from app")
		}),
	)

	app.Run()
}
