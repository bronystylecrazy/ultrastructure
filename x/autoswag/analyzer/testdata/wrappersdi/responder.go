package wrappersdi

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type jsonResponder struct{}

func (jsonResponder) Ok(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(web.Response{})
}
