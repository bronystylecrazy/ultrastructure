package migration

import (
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

type gooseZapLogger struct {
	logger *zap.SugaredLogger
}

var _ goose.Logger = (*gooseZapLogger)(nil)

// NewGooseZapLogger adapts zap.Logger to goose.Logger.
func NewGooseZapLogger(logger *zap.Logger) goose.Logger {
	if logger == nil {
		logger = zap.L()
	}

	return &gooseZapLogger{
		logger: logger.Sugar(),
	}
}

func (l *gooseZapLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatalf(format, v...)
}

func (l *gooseZapLogger) Printf(format string, v ...interface{}) {
	l.logger.Infof(format, v...)
}
