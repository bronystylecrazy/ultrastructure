package wrappersdi_ambig

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type responderA struct{}
type responderB struct{}

func NewResponderA() *responderA { return &responderA{} }
func NewResponderB() *responderB { return &responderB{} }

func (responderA) Ok(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(web.Response{})
}

func (responderB) Ok(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(web.Error{})
}
