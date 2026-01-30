package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

func AttachLoggerToOtel(base *zap.Logger, lp *LoggerProvider, config Config) *zap.Logger {
	if config.Disabled {
		return base
	}

	otelCore := otelzap.NewCore(config.Service, otelzap.WithLoggerProvider(lp))

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		base.Error("otel error", zap.Error(err))
	}))

	return base.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, otelCore)
	}))
}
