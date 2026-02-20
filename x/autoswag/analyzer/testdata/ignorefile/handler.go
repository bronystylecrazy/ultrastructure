package ignorefile

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

// @autoswag:ignore-file
type Handler struct{}

func (h *Handler) Handle(r web.Router) {
	r.Get("/ignored", h.Get)
}

func (h *Handler) Get(c fiber.Ctx) error {
	return c.SendString("ignored")
}
