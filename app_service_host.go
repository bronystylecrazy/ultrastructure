package us

import (
	"context"
	"time"

	kservice "github.com/kardianos/service"
	"go.uber.org/fx"
)

type appOption interface {
	apply(*App)
}

type serviceHostOption struct{}

// WithServiceHost enables wrapping App.Run with github.com/kardianos/service.
// Explicit CLI commands (for example `service`, `help`, `version`) still run in normal command mode.
func WithServiceHost() any {
	return serviceHostOption{}
}

func (serviceHostOption) apply(app *App) {
	if app == nil {
		return
	}
	app.enableServiceHost = true
}

type serviceHostProgram struct {
	owner *App
	app   *fx.App
}

func (p *serviceHostProgram) Start(_ kservice.Service) error {
	if p.owner == nil {
		return nil
	}
	fxApp := fx.New(p.owner.Build())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := fxApp.Start(ctx); err != nil {
		return err
	}
	p.app = fxApp
	return nil
}

func (p *serviceHostProgram) Stop(_ kservice.Service) error {
	if p.app == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return p.app.Stop(ctx)
}
