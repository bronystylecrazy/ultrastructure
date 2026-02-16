package web

import (
	"github.com/Flussen/swagger-fiber-v3"
	"github.com/gofiber/fiber/v3"
)

type SwaggerOption func(*swaggerOptions)

type SwaggerMiddleware struct {
	config Config
	path   string
}

func NewSwaggerMiddleware(config Config) (*SwaggerMiddleware, error) {
	return NewSwaggerMiddlewareWithOptions(WithSwaggerConfig(config))
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

func NewSwaggerMiddlewareWithOptions(opts ...SwaggerOption) (*SwaggerMiddleware, error) {
	cfg := swaggerOptions{path: "/docs/*"}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &SwaggerMiddleware{config: cfg.config, path: cfg.path}, nil
}

func (h *SwaggerMiddleware) Handle(r fiber.Router) {
	r.Get(h.path, swagger.HandlerDefault)
}
