package otel

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type obsKey struct{}
type loggerKey struct{}
type tracerKey struct{}

type Observer struct {
	*zap.Logger
	trace.Tracer
	otelmetric.Meter
	layerName              string
	isNop                  bool
	defaultMetricCtx       []attribute.KeyValue
	defaultAddMetricOpts   []otelmetric.AddOption
	defaultRecordMetricOpts []otelmetric.RecordOption
	instruments            *sync.Map
}

type Span struct {
	*Observer
	end func(options ...trace.SpanEndOption)
}

func NewObserver(logger *zap.Logger, tracer trace.Tracer, meter ...otelmetric.Meter) *Observer {
	m := metricnoop.NewMeterProvider().Meter("")
	if len(meter) > 0 && meter[0] != nil {
		m = meter[0]
	}
	return &Observer{
		Logger:                logger,
		Tracer:                tracer,
		Meter:                 m,
		defaultMetricCtx:      nil,
		defaultAddMetricOpts:  nil,
		defaultRecordMetricOpts: nil,
		instruments:           &sync.Map{},
	}
}

func NewNopObserver() *Observer {
	mp := metricnoop.NewMeterProvider()
	return &Observer{
		Logger:                zap.NewNop(),
		Tracer:                noop.NewTracerProvider().Tracer(""),
		Meter:                 mp.Meter(""),
		isNop:                 true,
		defaultMetricCtx:      nil,
		defaultAddMetricOpts:  nil,
		defaultRecordMetricOpts: nil,
		instruments:           &sync.Map{},
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

// ZapFieldsFromContext appends fields into the provided buffer and returns a slice
// of the populated entries. It avoids heap allocations when the caller reuses buf.
func ZapFieldsFromContext(ctx context.Context, buf *[4]zap.Field) []zap.Field {
	if ctx == nil || buf == nil {
		return nil
	}
	n := 0
	span := trace.SpanFromContext(ctx)
	if span == nil {
		if n == 0 {
			return nil
		}
		return buf[:n]
	}
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		if n == 0 {
			return nil
		}
		return buf[:n]
	}
	buf[n] = zap.String("trace.id", spanCtx.TraceID().String())
	n++
	buf[n] = zap.String("span.id", spanCtx.SpanID().String())
	n++
	buf[n] = zap.Bool("trace.sampled", spanCtx.IsSampled())
	n++
	return buf[:n]
}

func ContextFunc(ctx context.Context) []zapcore.Field {
	var buf [4]zap.Field
	return ZapFieldsFromContext(ctx, &buf)
}

// HasContext returns true if ctx contains observer, logger, or tracer overrides.
func HasContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if _, ok := ctx.Value(obsKey{}).(*Observer); ok {
		return true
	}
	if _, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return true
	}
	if _, ok := ctx.Value(tracerKey{}).(trace.Tracer); ok {
		return true
	}
	return false
}

// Span starts a new span using Observability from context (or a safe default).
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	return From(ctx).Span(ctx, name, opts...)
}

// Span starts a new span and returns enriched context plus span-scoped observability.
func (o *Observer) Span(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	if o == nil || o.Tracer == nil || o.Logger == nil {
		return From(ctx).Span(ctx, name, opts...)
	}
	ctx = o.With(ctx)
	ctx, span := o.Tracer.Start(ctx, name, opts...)

	// Enrich logger with trace context
	enrichedLogger := o.Logger
	if span.IsRecording() {
		fields := []zap.Field{
			zap.String("trace.id", span.SpanContext().TraceID().String()),
			zap.String("span.id", span.SpanContext().SpanID().String()),
			zap.Bool("trace.sampled", span.SpanContext().IsSampled()),
		}
		enrichedLogger = o.Logger.With(fields...)
	}

	// Store enriched obs back in context
	enrichedObs := &Observer{
		Logger:                 enrichedLogger,
		Tracer:                 o.Tracer,
		Meter:                  o.Meter,
		layerName:              o.layerName,
		defaultMetricCtx:       o.defaultMetricCtx,
		defaultAddMetricOpts:   o.defaultAddMetricOpts,
		defaultRecordMetricOpts: o.defaultRecordMetricOpts,
		instruments:            o.instruments,
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

func (o *Observer) AddCounter(ctx context.Context, name string, value int64, attrs ...attribute.KeyValue) {
	if o == nil || o.Meter == nil || name == "" {
		return
	}
	counter, err := o.int64Counter(name)
	if err != nil || counter == nil {
		return
	}
	counter.Add(ctx, value, o.withAddMetricAttributes(attrs)...)
}

func (o *Observer) AddFloatCounter(ctx context.Context, name string, value float64, attrs ...attribute.KeyValue) {
	if o == nil || o.Meter == nil || name == "" {
		return
	}
	counter, err := o.float64Counter(name)
	if err != nil || counter == nil {
		return
	}
	counter.Add(ctx, value, o.withAddMetricAttributes(attrs)...)
}

func (o *Observer) AddUpDownCounter(ctx context.Context, name string, value int64, attrs ...attribute.KeyValue) {
	if o == nil || o.Meter == nil || name == "" {
		return
	}
	counter, err := o.int64UpDownCounter(name)
	if err != nil || counter == nil {
		return
	}
	counter.Add(ctx, value, o.withAddMetricAttributes(attrs)...)
}

func (o *Observer) AddFloatUpDownCounter(ctx context.Context, name string, value float64, attrs ...attribute.KeyValue) {
	if o == nil || o.Meter == nil || name == "" {
		return
	}
	counter, err := o.float64UpDownCounter(name)
	if err != nil || counter == nil {
		return
	}
	counter.Add(ctx, value, o.withAddMetricAttributes(attrs)...)
}

func (o *Observer) RecordHistogram(ctx context.Context, name string, value float64, attrs ...attribute.KeyValue) {
	if o == nil || o.Meter == nil || name == "" {
		return
	}
	h, err := o.float64Histogram(name)
	if err != nil || h == nil {
		return
	}
	h.Record(ctx, value, o.withRecordMetricAttributes(attrs)...)
}

func (o *Observer) RecordIntHistogram(ctx context.Context, name string, value int64, attrs ...attribute.KeyValue) {
	if o == nil || o.Meter == nil || name == "" {
		return
	}
	h, err := o.int64Histogram(name)
	if err != nil || h == nil {
		return
	}
	h.Record(ctx, value, o.withRecordMetricAttributes(attrs)...)
}

func (o *Observer) withAddMetricAttributes(attrs []attribute.KeyValue) []otelmetric.AddOption {
	o.initDefaultMetricOptions()
	if len(attrs) == 0 {
		return o.defaultAddMetricOpts
	}
	merged := MergeMetricAttributes(o.defaultMetricCtx, attrs)
	if len(merged) == 0 {
		return nil
	}
	return []otelmetric.AddOption{
		otelmetric.WithAttributes(merged...),
	}
}

func (o *Observer) withRecordMetricAttributes(attrs []attribute.KeyValue) []otelmetric.RecordOption {
	o.initDefaultMetricOptions()
	if len(attrs) == 0 {
		return o.defaultRecordMetricOpts
	}
	merged := MergeMetricAttributes(o.defaultMetricCtx, attrs)
	if len(merged) == 0 {
		return nil
	}
	return []otelmetric.RecordOption{
		otelmetric.WithAttributes(merged...),
	}
}

func (o *Observer) initDefaultMetricOptions() {
	if o == nil || len(o.defaultMetricCtx) == 0 || o.defaultAddMetricOpts != nil || o.defaultRecordMetricOpts != nil {
		return
	}
	o.defaultAddMetricOpts = []otelmetric.AddOption{
		otelmetric.WithAttributes(o.defaultMetricCtx...),
	}
	o.defaultRecordMetricOpts = []otelmetric.RecordOption{
		otelmetric.WithAttributes(o.defaultMetricCtx...),
	}
}

func (o *Observer) int64Counter(name string) (otelmetric.Int64Counter, error) {
	key := "i64counter:" + name
	if v, ok := o.loadInstrument(key); ok {
		if instrument, ok := v.(otelmetric.Int64Counter); ok {
			return instrument, nil
		}
	}
	instrument, err := o.Meter.Int64Counter(name)
	if err != nil {
		return nil, err
	}
	if actual, loaded := o.storeInstrument(key, instrument); loaded {
		if cached, ok := actual.(otelmetric.Int64Counter); ok {
			return cached, nil
		}
	}
	return instrument, nil
}

func (o *Observer) float64Histogram(name string) (otelmetric.Float64Histogram, error) {
	key := "f64histogram:" + name
	if v, ok := o.loadInstrument(key); ok {
		if instrument, ok := v.(otelmetric.Float64Histogram); ok {
			return instrument, nil
		}
	}
	instrument, err := o.Meter.Float64Histogram(name)
	if err != nil {
		return nil, err
	}
	if actual, loaded := o.storeInstrument(key, instrument); loaded {
		if cached, ok := actual.(otelmetric.Float64Histogram); ok {
			return cached, nil
		}
	}
	return instrument, nil
}

func (o *Observer) int64Histogram(name string) (otelmetric.Int64Histogram, error) {
	key := "i64histogram:" + name
	if v, ok := o.loadInstrument(key); ok {
		if instrument, ok := v.(otelmetric.Int64Histogram); ok {
			return instrument, nil
		}
	}
	instrument, err := o.Meter.Int64Histogram(name)
	if err != nil {
		return nil, err
	}
	if actual, loaded := o.storeInstrument(key, instrument); loaded {
		if cached, ok := actual.(otelmetric.Int64Histogram); ok {
			return cached, nil
		}
	}
	return instrument, nil
}

func (o *Observer) float64Counter(name string) (otelmetric.Float64Counter, error) {
	key := "f64counter:" + name
	if v, ok := o.loadInstrument(key); ok {
		if instrument, ok := v.(otelmetric.Float64Counter); ok {
			return instrument, nil
		}
	}
	instrument, err := o.Meter.Float64Counter(name)
	if err != nil {
		return nil, err
	}
	if actual, loaded := o.storeInstrument(key, instrument); loaded {
		if cached, ok := actual.(otelmetric.Float64Counter); ok {
			return cached, nil
		}
	}
	return instrument, nil
}

func (o *Observer) int64UpDownCounter(name string) (otelmetric.Int64UpDownCounter, error) {
	key := "i64updown:" + name
	if v, ok := o.loadInstrument(key); ok {
		if instrument, ok := v.(otelmetric.Int64UpDownCounter); ok {
			return instrument, nil
		}
	}
	instrument, err := o.Meter.Int64UpDownCounter(name)
	if err != nil {
		return nil, err
	}
	if actual, loaded := o.storeInstrument(key, instrument); loaded {
		if cached, ok := actual.(otelmetric.Int64UpDownCounter); ok {
			return cached, nil
		}
	}
	return instrument, nil
}

func (o *Observer) float64UpDownCounter(name string) (otelmetric.Float64UpDownCounter, error) {
	key := "f64updown:" + name
	if v, ok := o.loadInstrument(key); ok {
		if instrument, ok := v.(otelmetric.Float64UpDownCounter); ok {
			return instrument, nil
		}
	}
	instrument, err := o.Meter.Float64UpDownCounter(name)
	if err != nil {
		return nil, err
	}
	if actual, loaded := o.storeInstrument(key, instrument); loaded {
		if cached, ok := actual.(otelmetric.Float64UpDownCounter); ok {
			return cached, nil
		}
	}
	return instrument, nil
}

func (o *Observer) loadInstrument(key string) (any, bool) {
	if o.instruments == nil {
		o.instruments = &sync.Map{}
	}
	return o.instruments.Load(key)
}

func (o *Observer) storeInstrument(key string, instrument any) (any, bool) {
	if o.instruments == nil {
		o.instruments = &sync.Map{}
	}
	return o.instruments.LoadOrStore(key, instrument)
}
