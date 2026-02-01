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

type Observer struct {
	*zap.Logger
	tracer trace.Tracer
}

type Span struct {
	*Observer
	end func(options ...trace.SpanEndOption)
}

func NewObserver(logger *zap.Logger, tracer trace.Tracer) *Observer {
	return &Observer{
		Logger: logger,
		tracer: tracer,
	}
}

// With stores Obs in context
func (o *Observer) With(ctx context.Context) context.Context {
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
func From(ctx context.Context) *Observer {
	if o, ok := ctx.Value(obsKey{}).(*Observer); ok {
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
	return NewObserver(logger, tracer)
}

// Span starts a new span using Observability from context (or a safe default).
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	return From(ctx).Span(ctx, name, opts...)
}

// Span starts a new span and returns enriched context plus span-scoped observability.
func (o *Observer) Span(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	ctx, span := o.tracer.Start(ctx, name, opts...)

	// Enrich logger with trace context
	enrichedLogger := o.Logger
	if span.IsRecording() {
		spanCtx := span.SpanContext()
		enrichedLogger = o.Logger.With(
			zap.String("trace.id", spanCtx.TraceID().String()),
			zap.Bool("trace.sampled", spanCtx.IsSampled()),
			zap.String("span.name", name),
			zap.String("span.id", spanCtx.SpanID().String()),
		)
	}

	// Store enriched obs back in context
	enrichedObs := &Observer{
		Logger: enrichedLogger,
		tracer: o.tracer,
	}
	ctx = enrichedObs.With(ctx)

	// Return context and span-scoped obs wrapper
	return ctx, &Span{
		Observer: enrichedObs,
		end:      span.End,
	}
}

// Start is a convenience alias for Span.
func (o *Observer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	return o.Span(ctx, name, opts...)
}

// End closes the span.
func (s *Span) End(options ...trace.SpanEndOption) {
	if s == nil || s.end == nil {
		return
	}
	s.end(options...)
}
