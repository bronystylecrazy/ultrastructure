package otel

import (
	"context"
	"os"

	"github.com/bronystylecrazy/ultrastructure/build"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func NewResource(ctx context.Context, config Config) (*resource.Resource, error) {

	var environment string
	if build.IsProduction() {
		environment = "development"
	} else {
		environment = "production"
	}

	hostName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.Service),
			semconv.ServiceNamespace(config.Namespace),
			semconv.DeploymentEnvironmentName(environment),
			semconv.HostName(hostName),
		),
	)
}
