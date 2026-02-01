package otel

import (
	"github.com/bronystylecrazy/ultrastructure/build"
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
			"trace.id",
			"span.id",
			"span.name",
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
		return zapcore.NewTee(c, otelzap.NewCore(config.Service, otelzap.WithLoggerProvider(lp)))
	})), nil
}

func NewBaseLogger(cfg Config) (*zap.Logger, error) {
	if build.IsDevelopment() {
		zapConfig := zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		switch cfg.Level {
		case "debug":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		case "info":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		case "warn":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
		case "error":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		case "fatal":
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
		default:
			zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		}
		return zapConfig.Build()
	}

	return zap.NewProduction()
}

func NewEventLogger(log *zap.Logger) fxevent.Logger {
	if build.IsProduction() {
		return fxevent.NopLogger
	}
	return &fxevent.ZapLogger{Logger: log}
}
