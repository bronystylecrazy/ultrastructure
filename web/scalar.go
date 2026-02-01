package web

import (
	"github.com/Flussen/swagger-fiber-v3"
	"github.com/bronystylecrazy/ultrastructure/di"
	_ "github.com/bronystylecrazy/ultrastructure/examples/otel-simple/docs"
	"github.com/gofiber/fiber/v3"
)

func UseScalar() di.Node {
	return di.Provide(NewScalarHandler)
}

type ScalarHandler struct {
	config Config
}

func NewScalarHandler(config Config) (*ScalarHandler, error) {
	return &ScalarHandler{config: config}, nil
}

func (h *ScalarHandler) Handle(r fiber.Router) {
	r.Get("/docs/*", swagger.HandlerDefault)
}
