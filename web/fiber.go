package web

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

type FiberConfig struct {
	Name string `mapstructure:"name"`
}

func NewFiberApp(config FiberConfig) *fiber.App {
	return fiber.New(fiber.Config{
		AppName: config.Name,
	})
}

func RegisterFiberApp(lc fx.Lifecycle, app *fiber.App, config Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go app.Listen(fmt.Sprintf("%s:%d", config.Host, config.Port))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	})
}
