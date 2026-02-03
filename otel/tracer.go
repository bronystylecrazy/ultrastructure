package otel

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

func NewTraceExporter(ctx context.Context, config Config, opts ...otlptracegrpc.Option) (*otlptrace.Exporter, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
		otlptracegrpc.WithTimeout(config.Timeout),
		otlptracegrpc.WithCompressor(config.Compressor),
	}

	for _, option := range opts {
		options = append(options, option)
	}

	// use Unstarted as the AutoGroup will handle the Start(ctx).
	return otlptracegrpc.NewUnstarted(
		options...,
	), nil
}
