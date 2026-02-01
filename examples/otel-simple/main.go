package main

import (
	"context"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/log"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type apiService struct {
	otel.Observable
	name string

	workerService *workerService
}

func NewAPIService(workerService *workerService) *apiService {
	return &apiService{name: "api", workerService: workerService}
}

func (s *apiService) Start(ctx context.Context) error {
	ctx, obs := s.Obs.Start(ctx, "service.start")
	defer obs.End()

	s.Obs.Info("Starting", zap.String("service", s.name))
	time.Sleep(20 * time.Millisecond)

	return s.workerService.Foo(ctx, "from api service")
}

func (s *apiService) Stop(ctx context.Context) error {
	ctx, obs := s.Obs.Start(ctx, "service.stop")
	defer obs.End()

	obs.Info("Stopping", zap.String("service", s.name))
	time.Sleep(10 * time.Millisecond)
	return nil
}

type workerService struct {
	otel.Observable
}

func NewWorkerService() *workerService {
	return &workerService{}
}

func (s *workerService) Foo(ctx context.Context, arg string) error {
	ctx, obs := s.Obs.Start(ctx, "worker.foo")
	defer obs.End()

	obs.Info("Foo called", zap.String("arg", arg))
	time.Sleep(5 * time.Millisecond)

	return s.Bar(ctx, arg)
}

func (s *workerService) Bar(ctx context.Context, arg string) error {
	ctx, obs := s.Obs.Start(ctx, "worker.bar")
	defer obs.End()

	obs.Info("Bar called", zap.String("arg", arg))
	time.Sleep(5 * time.Millisecond)
	return nil
}

func main() {
	options := di.App(
		log.Module(),
		otel.Module(),
		lifecycle.Module(),
		di.Provide(NewAPIService),
		di.Provide(NewWorkerService),
	).Build()

	fx.New(options).Run()
}
