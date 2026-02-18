package otel

import (
	"fmt"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/meta"
	xservice "github.com/bronystylecrazy/ultrastructure/service"
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

	wrapped, err = maybeAttachWindowsDaemonFileLog(wrapped)
	if err != nil {
		return nil, err
	}

	if !config.Enabled {
		return wrapped, nil
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		wrapped.Error("otel error", zap.Error(err))
	}))

	return wrapped.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, otelzap.NewCore(config.ServiceName, otelzap.WithLoggerProvider(lp)))
	})), nil
}

func maybeAttachWindowsDaemonFileLog(logger *zap.Logger) (*zap.Logger, error) {
	mode, err := xservice.RuntimeMode()
	if err != nil {
		return nil, fmt.Errorf("detect runtime mode: %w", err)
	}
	if mode != xservice.ModeDaemon {
		return logger, nil
	}

	logPath := xservice.WindowsServiceLogFile(meta.Name)
	if strings.TrimSpace(logPath) == "" {
		return logger, nil
	}

	fileCore, closeFn, err := openDaemonFileCore(logPath)
	if err != nil {
		return nil, fmt.Errorf("open daemon log file: %w", err)
	}
	_ = closeFn // process-lifetime file handle for daemon logging

	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, fileCore)
	})), nil
}

func NewBaseLogger(cfg Config) (*zap.Logger, error) {
	level := parseLogLevel(cfg.LogLevel)
	if !meta.IsProduction() {
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
	if meta.IsProduction() {
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
