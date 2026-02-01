package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	fiberzap "github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func NopZap() di.Node {
	return di.Replace(&ZapMiddleware{
		logger: zap.NewNop(),
	})
}

type ZapMiddleware struct {
	logger *zap.Logger
}

func NewZapMiddleware(logger *zap.Logger) *ZapMiddleware {
	return &ZapMiddleware{
		logger: logger,
	}
}

func (h *ZapMiddleware) Handle(r fiber.Router) {
	r.Use(fiberzap.New(fiberzap.Config{
		Logger: h.logger,
	}))
}
