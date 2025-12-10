package realtime

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Authorizer interface {
	Authorize() fiber.Handler
}

type authorizer struct {
	logger *zap.Logger
}

func NewAuthorizer(logger *zap.Logger) Authorizer {
	return &authorizer{logger: logger}
}

func (a *authorizer) Authorize() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
