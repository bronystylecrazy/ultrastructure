package web

import (
	"github.com/Flussen/swagger-fiber-v3"
	"github.com/bronystylecrazy/ultrastructure/di"
	_ "github.com/bronystylecrazy/ultrastructure/examples/otel-simple/docs"
	"github.com/gofiber/fiber/v3"
)

func UseSwagger() di.Node {
	return di.Provide(NewSwaggerHandler)
}

type SwaggerHandler struct {
	config Config
}

func NewSwaggerHandler(config Config) (*SwaggerHandler, error) {
	return &SwaggerHandler{config: config}, nil
}

func (h *SwaggerHandler) Handle(r fiber.Router) {
	r.Get("/docs/*", swagger.HandlerDefault)
}
