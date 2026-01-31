package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/log"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type service struct {
	name     string
	instance string
	id       string
}

func newService(name, instance string) *service {
	return &service{
		name:     name,
		instance: instance,
		id:       uuid.NewString(),
	}
}

type gatewayService struct{ *service }
type authService struct{ *service }
type catalogService struct{ *service }
type billingService struct{ *service }
type notificationsService struct{ *service }

func NewGatewayA() *gatewayService { return &gatewayService{service: newService("gateway", "a")} }
func NewGatewayB() *gatewayService { return &gatewayService{service: newService("gateway", "b")} }
func NewAuthA() *authService       { return &authService{service: newService("auth", "a")} }
func NewAuthB() *authService       { return &authService{service: newService("auth", "b")} }
func NewCatalogA() *catalogService { return &catalogService{service: newService("catalog", "a")} }
func NewCatalogB() *catalogService { return &catalogService{service: newService("catalog", "b")} }
func NewBillingA() *billingService { return &billingService{service: newService("billing", "a")} }
func NewBillingB() *billingService { return &billingService{service: newService("billing", "b")} }
func NewNotificationsA() *notificationsService {
	return &notificationsService{service: newService("notifications", "a")}
}
func NewNotificationsB() *notificationsService {
	return &notificationsService{service: newService("notifications", "b")}
}

func (s *service) Start(ctx context.Context) error {
	ctx, obs := otel.Start(ctx, "service.start")
	defer obs.End()

	obs.Info(
		"Starting service",
		zap.String("service", s.name),
		zap.String("instance", s.instance),
		zap.String("id", s.id),
	)

	s.simulateLifecycle(ctx, "startup")
	s.simulateTraffic(ctx)
	return nil
}

func (s *service) Stop(ctx context.Context) error {
	ctx, obs := otel.Start(ctx, "service.stop")
	defer obs.End()

	obs.Info(
		"Stopping service",
		zap.String("service", s.name),
		zap.String("instance", s.instance),
		zap.String("id", s.id),
	)

	s.simulateLifecycle(ctx, "shutdown")
	return nil
}

func (s *service) simulateLifecycle(ctx context.Context, phase string) {
	ctx, obs := otel.Start(ctx, fmt.Sprintf("service.%s", phase))
	defer obs.End()

	obs.Info(
		"Lifecycle phase",
		zap.String("phase", phase),
		zap.String("service", s.name),
		zap.String("instance", s.instance),
	)

	for i, step := range []string{"prepare", "warmup", "ready"} {

		stepCtx, stepObs := otel.Start(ctx, fmt.Sprintf("service.%s.%s", phase, step))
		stepObs.Info(
			"Lifecycle step",
			zap.Int("step", i+1),
			zap.String("phase", phase),
			zap.String("service", s.name),
			zap.String("instance", s.instance),
			zap.String("label", step),
		)

		_, innerObs := otel.Start(stepCtx, fmt.Sprintf("service.%s.%s.io", phase, step))
		innerObs.Info("Lifecycle io", zap.String("label", "io"))
		innerObs.End()

		time.Sleep(15 * time.Millisecond)
		stepObs.End()
	}
}

func (s *service) simulateTraffic(ctx context.Context) {
	for i := 0; i < 3; i++ {
		reqCtx, reqObs := otel.Start(ctx, fmt.Sprintf("service.%s.request", s.name))
		reqObs.Info(
			"Request",
			zap.Int("request", i+1),
			zap.String("service", s.name),
			zap.String("instance", s.instance),
		)

		s.simulateDependency(reqCtx, "cache")
		s.simulateDependency(reqCtx, "db")

		for _, target := range s.downstreamServices() {
			callCtx, callObs := otel.Start(reqCtx, fmt.Sprintf("service.call.%s", target))
			callObs.Info(
				"Outbound call",
				zap.String("service", s.name),
				zap.String("instance", s.instance),
				zap.String("target", target),
			)
			s.simulateDependency(callCtx, "http")
			callObs.End()
		}

		s.simulateDependency(reqCtx, "queue")
		reqObs.End()
	}
}

func (s *service) simulateDependency(ctx context.Context, name string) {
	depCtx, depObs := otel.Start(ctx, fmt.Sprintf("service.dep.%s", name))
	depObs.Info(
		"Dependency",
		zap.String("service", s.name),
		zap.String("instance", s.instance),
		zap.String("dependency", name),
	)
	_, innerObs := otel.Start(depCtx, fmt.Sprintf("service.dep.%s.io", name))
	innerObs.Info("Dependency io", zap.String("dependency", name))
	innerObs.End()
	time.Sleep(10 * time.Millisecond)
	depObs.End()
}

func (s *service) downstreamServices() []string {
	switch s.name {
	case "gateway":
		return []string{"auth", "catalog", "billing"}
	case "auth":
		return []string{"billing"}
	case "catalog":
		return []string{"billing"}
	case "billing":
		return []string{"notifications"}
	default:
		return nil
	}
}

func main() {
	options := di.App(
		log.Module(),
		otel.Module(),
		lifecycle.Module(),
		di.Provide(NewGatewayA, di.Name("gateway-a")),
		di.Provide(NewGatewayB, di.Name("gateway-b")),
		di.Provide(NewAuthA, di.Name("auth-a")),
		di.Provide(NewAuthB, di.Name("auth-b")),
		di.Provide(NewCatalogA, di.Name("catalog-a")),
		di.Provide(NewCatalogB, di.Name("catalog-b")),
		di.Provide(NewBillingA, di.Name("billing-a")),
		di.Provide(NewBillingB, di.Name("billing-b")),
		di.Provide(NewNotificationsA, di.Name("notifications-a")),
		di.Provide(NewNotificationsB, di.Name("notifications-b")),
	).Build()

	fx.New(options).Run()
}
