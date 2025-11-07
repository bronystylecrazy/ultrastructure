package logging

import (
	"github.com/bronystylecrazy/flexinfra/src/build"
	"github.com/bronystylecrazy/flexinfra/src/config"
	"github.com/bronystylecrazy/gx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(appConfig config.AppConfig) (*zap.Logger, error) {

	if build.IsDevelopmentMode() {
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

func NewFx(log *zap.Logger) fxevent.Logger {
	if gx.IsTestEnv() {
		return &fxevent.ZapLogger{Logger: zap.NewNop()}
	}
	return &fxevent.ZapLogger{Logger: log}
}
