package explicitlocal

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

func (h *Handler) Handle(r web.Router) {
	r.Get("/local", h.Get)
}

func (h *Handler) Get(c fiber.Ctx) error {
	return SendResponse(c)
}

func SendResponse(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(web.Response{})
}
