package otel

import (
	"log/slog"

	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SlogResult struct {
	fx.Out
	Handler slog.Handler
	Logger  *slog.Logger
}

func NewSlog(config Config, zapLogger *zap.Logger) SlogResult {
	handler := slogzap.Option{Level: ParseSlogLevel(config.LogLevel), Logger: zapLogger}.NewZapHandler()

	return SlogResult{
		Handler: handler,
		Logger:  slog.New(handler),
	}
}

func ParseSlogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
