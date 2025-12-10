package web

import (
	"github.com/gofiber/contrib/fiberzap"
	"go.uber.org/zap"
)

type LoggerHandler interface {
	Handler
}

var NopLoggerHandler LoggerHandler = &NopHandler{}

type loggerHandler struct {
	logger *zap.Logger
}

func NewLoggerHandler(logger *zap.Logger) LoggerHandler {
	return &loggerHandler{
		logger: logger,
	}
}

func (h *loggerHandler) Handle(app App) {
	app.Use(fiberzap.New(fiberzap.Config{
		Logger: h.logger,
	}))
}
