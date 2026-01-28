package log

import (
	"github.com/bronystylecrazy/ultrastructure/build"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(cfg Config) (*zap.Logger, error) {
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
	return &fxevent.ZapLogger{Logger: log}
}
