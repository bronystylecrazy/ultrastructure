package logging

import (
	"github.com/bronystylecrazy/flexinfra/build"
	"github.com/bronystylecrazy/flexinfra/config"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	SetLogger(logger *zap.Logger)
	L() *zap.Logger
}

var _ Logger = &Log{}

type Log struct {
	logger *zap.Logger
}

func (l *Log) SetLogger(logger *zap.Logger) {
	l.logger = logger
}

func (l *Log) L() *zap.Logger {
	if l.logger == nil {
		return NewDefaultLogger()
	}
	return l.logger
}

func NewZapLogger(appConfig config.AppConfig) (*zap.Logger, error) {

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

func NewDefaultLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func NewEventLogger(log *zap.Logger) fxevent.Logger {
	return &fxevent.ZapLogger{Logger: log}
}
