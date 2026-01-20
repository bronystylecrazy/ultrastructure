package us

import (
	"github.com/gofiber/fiber/v3"
)

func NewApp(cfg ...fiber.Config) fiber.Router {
	return fiber.New(cfg...)
}
