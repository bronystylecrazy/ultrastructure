package web

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bronystylecrazy/flexinfra/build"
	"github.com/bronystylecrazy/flexinfra/logging"
	"github.com/bronystylecrazy/flexinfra/realtime"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type FiberApp struct {
	*logging.Log
	*fiber.App

	rs     realtime.Server
	config Config
}

func NewFiberApp(config Config) App {
	return &FiberApp{
		App: fiber.New(fiber.Config{
			AppName:      build.Name,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  5 * time.Second,
			Network:      "tcp",
		}),
		config: config,
	}
}

func (f *FiberApp) Start(ctx context.Context) error {
	if f.rs != nil {
		if err := f.rs.Start(ctx); err != nil {
			return err
		}
	}

	go func() {
		if err := f.Listen(fmt.Sprintf(":%v", f.config.Port)); err != nil {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	return nil
}

func (f *FiberApp) Stop(ctx context.Context) error {
	if f.rs != nil {
		if err := f.rs.Stop(ctx); err != nil {
			return err
		}
	}
	return f.ShutdownWithContext(ctx)
}
