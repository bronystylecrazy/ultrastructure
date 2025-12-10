package logging

import "go.uber.org/zap"

func NewLogger(logger Logger) *zap.Logger {
	if logger.L() == nil {
		return NewDefaultLogger()
	}
	return logger.L()
}
