package web

import (
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v3"
)

type EtagHandler interface {
	Handler
}

var NopEtagHandler EtagHandler = NopHandler{}

type etagHandler struct {
}

func NewEtagHandler() EtagHandler {
	return &etagHandler{}
}

func (h *etagHandler) Handle(r fiber.Router) {
	r.Use(etag.New())
}
