package web

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bronystylecrazy/ultrastructure/build"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type fiberApp struct {
	*fiber.App

	rs     realtime.Server
	logger *zap.Logger
	config Config
}

func NewApp(config Config) App {
	return &fiberApp{
		App: fiber.New(fiber.Config{
			AppName:      build.Name,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  5 * time.Second,
			Network:      "tcp",
		}),
		config: config,
		logger: zap.NewNop(),
	}
}

func (f *fiberApp) SetLogger(logger *zap.Logger) {
	f.logger = logger
}

func (f *fiberApp) Start(ctx context.Context) error {
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

func (f *fiberApp) Stop(ctx context.Context) error {
	if f.rs != nil {
		if err := f.rs.Stop(ctx); err != nil {
			return err
		}
	}
	return f.ShutdownWithContext(ctx)
}
