package main

import (
	"github.com/bronystylecrazy/ultrastructure/core/logging"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HelloHandler struct{}

func NewHelloHandler() *HelloHandler {
	return &HelloHandler{}
}

func (h *HelloHandler) Handle(app web.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello from fiber")
	})
}

func main() {
	fx.New(
		fx.Provide(logging.NewDefaultLogger),
		fx.Supply(web.Config{
			Name:        "ultrastructure",
			Description: "fiber example",
			Host:        "localhost",
			Port:        "8080",
		}),
		web.Module(
			web.WithLogger(true),
			web.WithHealth(true),
			web.WithEtag(true),
			web.WithLimiter(true),
			web.WithSwagger(false),
			web.WithMonitor(false),
			web.WithStatic(false),
		),
		fx.Provide(web.AsHandler(NewHelloHandler)),
	).Run()
}
