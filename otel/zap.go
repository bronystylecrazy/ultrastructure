package otel

import (
	"strings"

	us "github.com/bronystylecrazy/ultrastructure"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(config Config, lp *LoggerProvider) (*zap.Logger, error) {
	base, err := NewBaseLogger(config)
	if err != nil {
		return nil, err
	}

	wrapped := base.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return FilterFieldsCore(
			c,
			"app.layer",
			"trace.id",
			"span.id",
			"trace.sampled",
		)
	}))

	if config.Disabled {
		return wrapped, nil
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		wrapped.Error("otel error", zap.Error(err))
	}))

	return wrapped.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, otelzap.NewCore(config.ServiceName, otelzap.WithLoggerProvider(lp)))
	})), nil
}

func NewBaseLogger(cfg Config) (*zap.Logger, error) {
	level := parseLogLevel(cfg.LogLevel)
	if !us.IsProduction() {
		zapConfig := zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.Level = zap.NewAtomicLevelAt(level)
		return zapConfig.Build()
	}

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(level)
	return zapConfig.Build()
}

func NewEventLogger(log *zap.Logger) fxevent.Logger {
	if us.IsProduction() {
		return fxevent.NopLogger
	}
	return &fxevent.ZapLogger{Logger: log}
}

func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
