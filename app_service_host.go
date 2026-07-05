package us

import (
	"context"
	"os"
	"sync/atomic"
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
	owner    *App
	app      *fx.App
	stopping atomic.Bool
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
	go p.watchShutdown(fxApp)
	return nil
}

// watchShutdown exits the process when the app shuts itself down (for example
// via fx.Shutdowner after a command finishes); the service host would
// otherwise keep a stopped app registered as running forever.
func (p *serviceHostProgram) watchShutdown(fxApp *fx.App) {
	sig := <-fxApp.Wait()
	if !p.stopping.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	_ = fxApp.Stop(ctx)
	cancel()
	os.Exit(sig.ExitCode)
}

func (p *serviceHostProgram) Stop(_ kservice.Service) error {
	if p.app == nil {
		return nil
	}
	if !p.stopping.CompareAndSwap(false, true) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return p.app.Stop(ctx)
}
