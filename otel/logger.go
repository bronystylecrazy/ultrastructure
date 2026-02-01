package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
)

func NewLogExporter(ctx context.Context, config Config, opts ...otlploggrpc.Option) (*otlploggrpc.Exporter, error) {
	options := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(config.Endpoint),
		otlploggrpc.WithTimeout(config.Timeout),
		otlploggrpc.WithCompressor(config.Compressor),
	}

	if config.AuthKey == "" {
		options = append(options, otlploggrpc.WithInsecure())
	} else {
		options = append(options, otlploggrpc.WithHeaders(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %v", config.AuthKey),
		}))
	}

	for _, option := range opts {
		options = append(options, option)
	}

	return otlploggrpc.New(ctx, options...)
}
