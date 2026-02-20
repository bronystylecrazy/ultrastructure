package wrappers

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappers/resp"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	responder resp.Responder
}

func (h *Handler) Handle(r web.Router) {
	r.Get("/wrapped-func", h.WrappedFunc)
	r.Get("/wrapped-method", h.WrappedMethod)
}

func (h *Handler) WrappedFunc(c fiber.Ctx) error {
	return resp.Ok(c)
}

func (h *Handler) WrappedMethod(c fiber.Ctx) error {
	return h.responder.Created(c)
}
