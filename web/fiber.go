package web

import (
	"context"
	"fmt"
	"time"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FiberConfig struct {
	Name string `mapstructure:"name"`
}

func NewFiberApp(config FiberConfig) *fiber.App {
	return fiber.New(fiber.Config{
		AppName:      buildAppName(config.Name),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		IdleTimeout:  2 * time.Second,
	})
}

func buildAppName(name string) string {
	if name == "" {
		name = "app"
	}
	return fmt.Sprintf("%s (%s %s %s)", name, us.Version, us.Commit, us.BuildDate)
}

func RegisterFiberApp(lc fx.Lifecycle, app *fiber.App, logger *zap.Logger, config Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				err := app.Listen(fmt.Sprintf("%s:%d", config.Host, config.Port))
				if err != nil {
					logger.Error("failed to start fiber app", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	})
}
