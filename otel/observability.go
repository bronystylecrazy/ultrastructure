package otel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type obsKey struct{}
type loggerKey struct{}
type tracerKey struct{}

type Observability struct {
	*zap.Logger
	tracer trace.Tracer
}

type Span struct {
	*Observability
	end func(options ...trace.SpanEndOption)
}

func NewObservability(logger *zap.Logger, tracer trace.Tracer) *Observability {
	return &Observability{
		Logger: logger,
		tracer: tracer,
	}
}

// With stores Obs in context
func (o *Observability) With(ctx context.Context) context.Context {
	return context.WithValue(ctx, obsKey{}, o)
}

// WithLogger stores a zap.Logger in context for later span creation.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}

// WithTracer stores a trace.Tracer in context for later span creation.
func WithTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	if tracer == nil {
		return ctx
	}
	return context.WithValue(ctx, tracerKey{}, tracer)
}

// From retrieves Obs from context
func From(ctx context.Context) *Observability {
	if o, ok := ctx.Value(obsKey{}).(*Observability); ok {
		return o
	}

	logger, _ := ctx.Value(loggerKey{}).(*zap.Logger)
	tracer, _ := ctx.Value(tracerKey{}).(trace.Tracer)
	if logger == nil {
		logger = zap.NewNop()
	}
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("")
	}
	// Return safe default (or context-provided logger/tracer)
	return NewObservability(logger, tracer)
}

// Span starts a new span using Observability from context (or a safe default).
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	return From(ctx).Span(ctx, name, opts...)
}

// Span starts a new span and returns enriched context plus span-scoped observability.
func (o *Observability) Span(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	ctx, span := o.tracer.Start(ctx, name, opts...)

	// Enrich logger with trace context
	enrichedLogger := o.Logger
	if span.IsRecording() {
		spanCtx := span.SpanContext()
		enrichedLogger = o.Logger.With(
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.Bool("trace_sampled", spanCtx.IsSampled()),
			zap.String("span_name", name),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Store enriched obs back in context
	enrichedObs := &Observability{
		Logger: enrichedLogger,
		tracer: o.tracer,
	}
	ctx = enrichedObs.With(ctx)

	// Return context and span-scoped obs wrapper
	return ctx, &Span{
		Observability: enrichedObs,
		end:           span.End,
	}
}

// End closes the span.
func (s *Span) End(options ...trace.SpanEndOption) {
	if s == nil || s.end == nil {
		return
	}
	s.end(options...)
}
