package otel

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

type LoggerProvider struct {
	*sdklog.LoggerProvider
}

func NewLoggerProvider(ctx context.Context, resource *resource.Resource, exporter *otlploggrpc.Exporter) (*LoggerProvider, error) {
	return &LoggerProvider{sdklog.NewLoggerProvider(
		sdklog.WithResource(resource),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)}, nil
}

func (lp *LoggerProvider) Stop(ctx context.Context) error {
	return lp.Shutdown(ctx)
}
