// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package aws provides instrumentation for the AWS SDK.
package aws // import "github.com/bronystylecrazy/ultrastructure/otel/aws"

import (
	"context"
	"time"

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	usotel "github.com/bronystylecrazy/ultrastructure/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ScopeName is the instrumentation scope name.
	ScopeName = "github.com/bronystylecrazy/ultrastructure/otel/aws"
)

type spanTimestampKey struct{}

// AttributeBuilder returns an array of KeyValue pairs, it can be used to set custom attributes.
type AttributeBuilder func(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) []attribute.KeyValue

type OtelMiddlewares struct {
	usotel.Telemetry
	tracer            trace.Tracer
	propagator        propagation.TextMapPropagator
	attributeBuilders []AttributeBuilder
}

func (m *OtelMiddlewares) initializeMiddlewareBefore(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OTelInitializeMiddlewareBefore", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error,
	) {
		ctx = context.WithValue(ctx, spanTimestampKey{}, time.Now())
		return next.HandleInitialize(ctx, in)
	}),
		middleware.Before)
}

func (m *OtelMiddlewares) initializeMiddlewareAfter(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("OTelInitializeMiddlewareAfter", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error,
	) {
		serviceID := v2Middleware.GetServiceID(ctx)
		operation := v2Middleware.GetOperationName(ctx)
		region := v2Middleware.GetRegion(ctx)

		attributes := []attribute.KeyValue{
			SystemAttr(),
			MethodAttr(serviceID, operation),
			RegionAttr(region),
		}

		end := func(...trace.SpanEndOption) {}
		if m.Obs != nil {
			var span *usotel.Span
			ctx, span = m.Obs.Start(ctx, spanName(serviceID, operation),
				trace.WithTimestamp(ctx.Value(spanTimestampKey{}).(time.Time)),
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(attributes...),
			)
			end = span.End
		} else {
			var span trace.Span
			ctx, span = m.tracer.Start(ctx, spanName(serviceID, operation),
				trace.WithTimestamp(ctx.Value(spanTimestampKey{}).(time.Time)),
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(attributes...),
			)
			end = span.End
		}
		defer end()

		out, metadata, err = next.HandleInitialize(ctx, in)
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(m.buildAttributes(ctx, in, out)...)
		if err != nil {
			span.SetAttributes(semconv.ErrorType(err))
			span.SetStatus(codes.Error, err.Error())
		}

		return out, metadata, err
	}),
		middleware.After)
}

func (m *OtelMiddlewares) finalizeMiddlewareAfter(stack *middleware.Stack) error {
	return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("OTelFinalizeMiddleware", func(
		ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (
		out middleware.FinalizeOutput, metadata middleware.Metadata, err error,
	) {
		// Propagate the Trace information by injecting it into the HTTP request.
		switch req := in.Request.(type) {
		case *smithyhttp.Request:
			m.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
		default:
		}

		return next.HandleFinalize(ctx, in)
	}),
		middleware.After)
}

func (*OtelMiddlewares) deserializeMiddleware(stack *middleware.Stack) error {
	return stack.Deserialize.Add(middleware.DeserializeMiddlewareFunc("OTelDeserializeMiddleware", func(
		ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (
		out middleware.DeserializeOutput, metadata middleware.Metadata, err error,
	) {
		out, metadata, err = next.HandleDeserialize(ctx, in)
		resp, ok := out.RawResponse.(*smithyhttp.Response)
		if !ok {
			// No raw response to wrap with.
			return out, metadata, err
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(semconv.HTTPResponseStatusCode(resp.StatusCode))

		requestID, ok := v2Middleware.GetRequestIDMetadata(metadata)
		if ok {
			span.SetAttributes(RequestIDAttr(requestID))
		}

		return out, metadata, err
	}),
		middleware.Before)
}

func (m *OtelMiddlewares) buildAttributes(ctx context.Context, in middleware.InitializeInput, out middleware.InitializeOutput) (attributes []attribute.KeyValue) {
	for _, builder := range m.attributeBuilders {
		attributes = append(attributes, builder(ctx, in, out)...)
	}

	return attributes
}

func spanName(serviceID, operation string) string {
	spanName := serviceID
	if operation != "" {
		spanName += "." + operation
	}
	return spanName
}

func NewMiddlewares() *OtelMiddlewares {
	cfg := config{
		TracerProvider:    otel.GetTracerProvider(),
		TextMapPropagator: otel.GetTextMapPropagator(),
	}

	if cfg.AttributeBuilders == nil {
		cfg.AttributeBuilders = []AttributeBuilder{DefaultAttributeBuilder}
	}

	return &OtelMiddlewares{
		tracer: cfg.TracerProvider.Tracer(ScopeName,
			trace.WithInstrumentationVersion(Version)),
		Telemetry:         usotel.Nop(),
		propagator:        cfg.TextMapPropagator,
		attributeBuilders: cfg.AttributeBuilders,
	}
}

func (m *OtelMiddlewares) Append(apiOptions *[]func(*middleware.Stack) error) {
	*apiOptions = append(
		*apiOptions,
		m.initializeMiddlewareBefore,
		m.initializeMiddlewareAfter,
		m.finalizeMiddlewareAfter,
		m.deserializeMiddleware,
	)
}
