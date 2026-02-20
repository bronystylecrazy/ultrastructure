package resp

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Responder struct{}

func Ok(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(web.Response{})
}

func (Responder) Created(c fiber.Ctx) error {
	return c.Status(fiber.StatusCreated).JSON(web.Response{})
}
