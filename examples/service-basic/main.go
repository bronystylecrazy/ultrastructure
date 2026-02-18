package main

import (
	us "github.com/bronystylecrazy/ultrastructure"
	xcmd "github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Handle(r web.Router) {
	r.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}

func main() {
	us.New(
		us.WithServiceHost(),
		xcmd.UseBasicCommands(),
		xcmd.UseServiceCommands(),
		xcmd.Run(
			web.Init(),
			di.Provide(NewHealthHandler, di.As[web.Handler]()),
		),
	).Run()
}
