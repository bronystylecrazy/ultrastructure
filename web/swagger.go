package web

import (
	"github.com/Flussen/swagger-fiber-v3"
	"github.com/gofiber/fiber/v3"
)

type SwaggerOption func(*swaggerOptions)

type SwaggerHandler struct {
	config Config
	path   string
}

func NewSwaggerHandler(config Config) (*SwaggerHandler, error) {
	return NewSwaggerHandlerWithOptions(WithSwaggerConfig(config))
}

type swaggerOptions struct {
	config Config
	path   string
}

func WithSwaggerConfig(config Config) SwaggerOption {
	return func(o *swaggerOptions) {
		o.config = config
	}
}

func WithSwaggerPath(path string) SwaggerOption {
	return func(o *swaggerOptions) {
		o.path = path
	}
}

func NewSwaggerHandlerWithOptions(opts ...SwaggerOption) (*SwaggerHandler, error) {
	cfg := swaggerOptions{path: "/docs/*"}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &SwaggerHandler{config: cfg.config, path: cfg.path}, nil
}

func (h *SwaggerHandler) Handle(r fiber.Router) {
	r.Get(h.path, swagger.HandlerDefault)
}
