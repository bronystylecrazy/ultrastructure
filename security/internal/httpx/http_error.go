package common

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func Unauthorized(c fiber.Ctx, message string) error {
	if message == "" {
		message = "unauthorized"
	}
	return c.Status(fiber.StatusUnauthorized).JSON(web.Error{
		Error: web.ErrorDetail{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	})
}

func Forbidden(c fiber.Ctx, message string) error {
	if message == "" {
		message = "forbidden"
	}
	return c.Status(fiber.StatusForbidden).JSON(web.Error{
		Error: web.ErrorDetail{
			Code:    "FORBIDDEN",
			Message: message,
		},
	})
}
