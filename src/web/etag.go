package web

import (
	"github.com/gofiber/fiber/v2/middleware/etag"
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

func (h *etagHandler) Handle(app Router) {
	app.Use(etag.New())
}
