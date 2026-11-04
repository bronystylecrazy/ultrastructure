package rd

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

// childOnlyTracerProvider hands out tracers that start a span only under a
// valid parent span context, so that a command issued outside any trace, such
// as one from a background loop, does not open a trace of its own.
type childOnlyTracerProvider struct {
	embedded.TracerProvider
	tp trace.TracerProvider
}

func (p childOnlyTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return childOnlyTracer{Tracer: p.tp.Tracer(name, opts...)}
}

type childOnlyTracer struct {
	trace.Tracer
}

func (t childOnlyTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return ctx, noop.Span{}
	}
	return t.Tracer.Start(ctx, name, opts...)
}
