package otel

import (
	"context"
	"math"
	"strings"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func buildSamplerConfigFrom(config Config) sdktrace.Sampler {
	name := strings.ToLower(strings.TrimSpace(config.Traces.Sampler))
	ratio := config.Traces.SamplerArg
	if math.IsNaN(ratio) || ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	switch name {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(ratio)
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}

type TracerProvider struct {
	*sdktrace.TracerProvider
}

func NewTracerProvider(ctx context.Context, resource *resource.Resource, config Config, exporter sdktrace.SpanExporter) (*TracerProvider, error) {
	if exporter == nil {
		return &TracerProvider{sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.NeverSample()),
			sdktrace.WithResource(resource),
		)}, nil
	}
	sampler := buildSamplerConfigFrom(config)
	return &TracerProvider{sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(resource),
	)}, nil
}

func (tp *TracerProvider) Stop(ctx context.Context) error {
	return tp.Shutdown(ctx)
}
