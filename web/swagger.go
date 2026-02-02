package web

import (
	"github.com/Flussen/swagger-fiber-v3"
	_ "github.com/bronystylecrazy/ultrastructure/examples/otel-simple/docs"
	"github.com/gofiber/fiber/v3"
)

type SwaggerHandler struct {
	config Config
}

func NewSwaggerHandler(config Config) (*SwaggerHandler, error) {
	return &SwaggerHandler{config: config}, nil
}

func (h *SwaggerHandler) Handle(r fiber.Router) {
	r.Get("/docs/*", swagger.HandlerDefault)
}
