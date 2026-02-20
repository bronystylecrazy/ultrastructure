package explicitgeneric

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

func (h *Handler) Handle(r web.Router) {
	r.Get("/generic", h.Get)
}

func (h *Handler) Get(c fiber.Ctx) error {
	return SendResponse[web.Response](c, web.Response{})
}

func SendResponse[T any](c fiber.Ctx, payload T) error {
	return c.Status(fiber.StatusOK).JSON(payload)
}
