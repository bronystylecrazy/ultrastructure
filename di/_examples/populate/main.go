package main

import (
	"context"
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AppConfig struct {
	Name string
	Port int
}

func main() {
	var (
		logger *zap.Logger
		cfg    AppConfig
	)

	app := fx.New(
		di.App(
			di.Provide(zap.NewDevelopment, di.Name("dev")),
			di.Supply(AppConfig{Name: "demo", Port: 9000}),
			di.Populate(&logger, di.Name("dev")),
			di.Populate(&cfg),
		).Build(),
	)
	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = app.Stop(context.Background()) }()
	logger.Info("populated", zap.String("name", cfg.Name), zap.Int("port", cfg.Port))
}
