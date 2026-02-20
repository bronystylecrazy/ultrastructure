package wrappersdi_ambig

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Responder interface {
	Ok(c fiber.Ctx) error
}

type handler struct {
	responder Responder
}

func NewHandler(responder Responder) *handler {
	return &handler{responder: responder}
}

func Use() di.Node {
	return di.Options(
		di.Provide(NewResponderA),
		di.Provide(NewResponderB),
		di.Provide(NewHandler),
	)
}

func (h *handler) Handle(r web.Router) {
	r.Get("/di-ambig", h.Wrapped)
}

func (h *handler) Wrapped(c fiber.Ctx) error {
	return h.responder.Ok(c)
}
