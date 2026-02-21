package headers

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func Use() any {
	return NewHandler()
}

func (h *Handler) Handle(r web.Router) {
	r.Get("/headers/:id", h.Get)
}

func (h *Handler) Get(c fiber.Ctx) error {
	c.Set("X-Request-ID", "req-123")
	c.Cookie(&fiber.Cookie{Name: "session_id", Value: "abc"})
	return c.Status(fiber.StatusCreated).JSON(web.Response{})
}
