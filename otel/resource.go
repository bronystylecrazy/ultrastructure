package otel

import (
	"context"
	"os"

	us "github.com/bronystylecrazy/ultrastructure"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func NewResource(ctx context.Context, config Config) (*resource.Resource, error) {

	var environment string
	if us.IsProduction() {
		environment = "development"
	} else {
		environment = "production"
	}

	hostName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	attrs := []attribute.KeyValue{
		semconv.HostName(hostName),
		semconv.DeploymentEnvironmentName(environment),
	}
	if config.ServiceName != "" {
		attrs = append(attrs, semconv.ServiceName(config.ServiceName))
	}
	if len(config.ResourceAttrs) > 0 {
		for k, v := range config.ResourceAttrs {
			attrs = append(attrs, attribute.String(k, v))
		}
	}

	return resource.New(
		ctx,
		resource.WithAttributes(attrs...),
	)
}
