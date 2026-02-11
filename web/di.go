package web

import (
	"embed"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"go.uber.org/zap"
)

func RunFiberApp() di.Node {
	return di.Options(
		di.Invoke(SetupHandlers),
		di.Invoke(RegisterFiberApp),
	)
}

func UseOtel() di.Node {
	return di.Provide(NewOtelMiddleware, otel.Layer("http"), Priority(Earliest))
}

func UseSwagger(opts ...SwaggerOption) di.Node {
	return di.Options(
		di.Provide(func(config Config) (*SwaggerHandler, error) {
			base := []SwaggerOption{
				WithSwaggerConfig(config),
			}
			return NewSwaggerHandlerWithOptions(append(base, opts...)...)
		}),
	)
}

func UseSpa(opts ...SpaOption) di.Node {
	return di.Options(
		di.Provide(func(assets *embed.FS, log *zap.Logger) (*SpaMiddleware, error) {
			base := []SpaOption{
				WithSpaAssets(assets),
				WithSpaLogger(log),
			}
			return NewSpaMiddlewareWithOptions(append(base, opts...)...)
		}, di.Params(di.Optional()), Priority(Latest)),
	)
}
