package otel

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type TracerProvider struct {
	*sdktrace.TracerProvider
}

func NewTracerProvider(ctx context.Context, resource *resource.Resource, exporter *otlptrace.Exporter) (*TracerProvider, error) {
	return &TracerProvider{sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)}, nil
}

func (tp *TracerProvider) Stop(ctx context.Context) error {
	return tp.Shutdown(ctx)
}
