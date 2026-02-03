package buildinfo

import "github.com/gofiber/fiber/v3"

func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		defaultPath: DefaultPath,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

func (h *Handler) Handle(r fiber.Router) {
	r.Get(h.defaultPath, func(c fiber.Ctx) error {
		return c.SendString("Build info")
	})
}
