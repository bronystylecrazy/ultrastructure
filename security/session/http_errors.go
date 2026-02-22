package session

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func writeUnauthorized(c fiber.Ctx, err error) error {
	msg := "unauthorized"
	if err != nil && err.Error() != "" {
		msg = err.Error()
	}
	return c.Status(fiber.StatusUnauthorized).JSON(web.Error{
		Error: web.ErrorDetail{
			Code:    "UNAUTHORIZED",
			Message: msg,
		},
	})
}
