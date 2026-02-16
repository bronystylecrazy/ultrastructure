package otel

import (
	"context"
	"os"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/meta"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func NewResource(ctx context.Context, config Config) (*resource.Resource, error) {
	environment := ""
	if len(config.ResourceAttrs) > 0 {
		environment = strings.TrimSpace(config.ResourceAttrs["deployment.environment"])
	}
	if environment == "" {
		if meta.IsProduction() {
			environment = "production"
		} else {
			environment = "development"
		}
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
			if k == "deployment.environment" {
				// Avoid duplicate semconv/resource attribute key; semconv value above is canonical.
				continue
			}
			attrs = append(attrs, attribute.String(k, v))
		}
	}

	return resource.New(
		ctx,
		resource.WithAttributes(attrs...),
	)
}
