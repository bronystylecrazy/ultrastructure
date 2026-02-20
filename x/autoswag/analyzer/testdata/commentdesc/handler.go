package commentdesc

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

func (h *Handler) Handle(r web.Router) {
	r.Get("/comment-desc", func(c fiber.Ctx) error {
		/*
			This is example command
		*/
		return c.SendString("ok")
	})

	r.Get("/comment-inline", func(c fiber.Ctx) error {
		return c.SendString("ok") // Inline comment description
	})
}
