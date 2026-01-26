package us

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

package logging

import (
	"github.com/bronystylecrazy/ultrastructure/build"
	"github.com/bronystylecrazy/ultrastructure/config"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() (*zap.Logger, error) {

	if build.IsDevelopment() {
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		switch appConfig.LogLevel {
		case "debug":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		case "info":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		case "warn":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
		case "error":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		case "fatal":
			cfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
		default:
			cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		}
		logger, err := cfg.Build()
		if err != nil {
			return nil, err
		}
		return logger, nil
	}

	return zap.NewProduction()
}

func NewEventLogger(log *zap.Logger) fxevent.Logger {
	return &fxevent.ZapLogger{Logger: log}
}
