package web

import (
	"fmt"

	"github.com/arsmn/fiber-swagger/example/docs"
	"github.com/bronystylecrazy/flexinfra/src/config"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type SwaggerHandler interface {
	Handler
}

var NopSwaggerHandler SwaggerHandler = &NopHandler{}

type swaggerHandler struct {
	appConfig config.AppConfig
}

func NewSwaggerHandler(appConfig config.AppConfig) SwaggerHandler {
	return &swaggerHandler{appConfig: appConfig}
}

func (h *swaggerHandler) Handle(app Router) {
	docs.SwaggerInfo.Title = h.appConfig.Name
	docs.SwaggerInfo.Description = h.appConfig.Description
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%v", h.appConfig.Port)
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Version = "1.0"
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
}
