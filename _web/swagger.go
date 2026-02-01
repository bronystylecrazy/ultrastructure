package web

import (
	"fmt"

	"github.com/arsmn/fiber-swagger/example/docs"
	"github.com/bronystylecrazy/ultrastructure/build"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type SwaggerHandler interface {
	Handler
}

var NopSwaggerHandler SwaggerHandler = &NopHandler{}

type swaggerHandler struct {
	config Config
}

func NewSwaggerHandler(config Config) (SwaggerHandler, error) {
	return &swaggerHandler{config: config}, nil
}

func (h *swaggerHandler) Handle(app App) {
	docs.SwaggerInfo.Title = h.config.Name
	docs.SwaggerInfo.Description = h.config.Description
	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%s", h.config.Host, h.config.Port)
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Version = build.Version
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
}
