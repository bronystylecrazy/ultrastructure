package wrappersdi_unresolved

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Responder interface {
	Ok(c fiber.Ctx) error
}

type handler struct {
	responder Responder
}

func (h *handler) Handle(r web.Router) {
	r.Get("/di-unresolved", h.Wrapped)
}

func (h *handler) Wrapped(c fiber.Ctx) error {
	return h.responder.Ok(c)
}
