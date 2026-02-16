package web

import (
	"embed"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func IgnoreAutoGroupHandlers() di.Option {
	return di.AutoGroupIgnoreType[Handler](HandlersGroupName)
}

func Init() di.Node {
	return di.Options(
		di.Invoke(func(router fiber.Router, otelMiddleware *OtelMiddleware) {
			otelMiddleware.Handle(router)
		}, di.Params(di.Optional(), di.Optional())),
		di.Invoke(func(router fiber.Router, swaggerMiddleware *SwaggerMiddleware) {
			if swaggerMiddleware == nil {
				return
			}
			swaggerMiddleware.Handle(router)
		}, di.Params(di.Optional(), di.Optional())),
		di.Invoke(func(router fiber.Router, spaMiddleware *SpaMiddleware) {
			if spaMiddleware == nil {
				return
			}
			spaMiddleware.Handle(router)
		}, di.Params(di.Optional(), di.Optional())),
		di.Invoke(SetupHandlers),
		di.Invoke(RegisterFiberApp),
	)
}

func UseSwagger(opts ...SwaggerOption) di.Node {
	return di.Provide(func(config Config) (*SwaggerMiddleware, error) {
		base := []SwaggerOption{
			WithSwaggerConfig(config),
		}
		return NewSwaggerMiddlewareWithOptions(append(base, opts...)...)
	}, IgnoreAutoGroupHandlers())
}

func UseSpa(opts ...SpaOption) di.Node {
	return di.Provide(func(assets *embed.FS, log *zap.Logger) (*SpaMiddleware, error) {
		base := []SpaOption{
			WithSpaAssets(assets),
			WithSpaLogger(log),
		}
		return NewSpaMiddlewareWithOptions(append(base, opts...)...)
	}, IgnoreAutoGroupHandlers(), di.Params(di.Optional()), Priority(Latest))
}
