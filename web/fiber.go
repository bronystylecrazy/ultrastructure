package web

import "github.com/gofiber/fiber/v3"

type FiberConfig struct {
	Name string `mapstructure:"name"`
}

func NewFiberApp(config FiberConfig) *fiber.App {
	return fiber.New(fiber.Config{
		AppName: config.Name,
	})
}
