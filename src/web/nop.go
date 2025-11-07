package web

import "github.com/gofiber/fiber/v2"

type NopHandler struct{}

func (n NopHandler) Handle(app Router) {}

type NopAuthorizer struct{}

func (n NopAuthorizer) Authorize() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
