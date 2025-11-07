package web

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bronystylecrazy/flexinfra/src/build"
	"github.com/bronystylecrazy/flexinfra/src/config"
	"github.com/bronystylecrazy/flexinfra/src/realtime"
	"github.com/bronystylecrazy/gx"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewApp(logger *zap.Logger) Router {
	return fiber.New(fiber.Config{
		AppName:      build.Name,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
		Network:      "tcp",
	})
}

func StartApp(
	lc fx.Lifecycle,
	appConfig config.AppConfig,
	app fiber.Router,
	realtimeServer realtime.Server,
	log *zap.Logger,
) error {
	fiberApp, ok := app.(*fiber.App)
	if !ok {
		return errors.New("failed to cast app to *fiber.App")
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if err := realtimeServer.Start(); err != nil {
				return err
			}
			go func() {
				if !gx.IsTestEnv() {
					if err := fiberApp.Listen(fmt.Sprintf(":%v", appConfig.Port)); err != nil {
						log.Fatal("Failed to start server", zap.Error(err))
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := realtimeServer.Close(); err != nil {
				return err
			}
			return fiberApp.ShutdownWithContext(ctx)
		},
	})

	return nil
}
