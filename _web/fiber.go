package web

import "github.com/gofiber/fiber/v3"

type Config struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

func NewFiberApp(config ...fiber.Config) *fiber.App {
	return fiber.New(config...)
}
