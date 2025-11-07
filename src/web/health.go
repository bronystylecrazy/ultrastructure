package web

import "github.com/gofiber/fiber/v2/middleware/healthcheck"

type HealthHandler interface {
	Handler
}

var NopHealthHandler HealthHandler = &NopHandler{}

type healthHandler struct {
}

func NewHealthHandler() HealthHandler {
	return &healthHandler{}
}

func (h *healthHandler) Handle(app Router) {
	app.Use(healthcheck.New())
}
