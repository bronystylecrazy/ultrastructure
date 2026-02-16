package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/gofiber/fiber/v3"
)

var HandlersGroupName = "us.handlers"

func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.Module(
			"us.web",
			di.AutoGroup[SwaggerCustomizer](SwaggerCustomizersGroupName),
			di.AutoGroup[SwaggerPreRun](SwaggerPreCustomizersGroupName),
			di.AutoGroup[SwaggerPostRun](SwaggerPostCustomizersGroupName),

			di.Config[Config]("web"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),

			di.Config[FiberConfig]("web.fiber"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
			di.Provide(NewRegistryContainer),
			di.Provide(NewSwaggerModelRegistry),
			di.Provide(NewRegistryLifecycle),
			di.Provide(NewFiberApp, di.AsSelf[fiber.Router]()),
			di.Provide(NewOtelMiddleware, otel.Layer("web.http"), IgnoreAutoGroupHandlers(), Priority(Earliest)),
			di.Options(di.ConvertAnys(extends)...),
		),
	)
}
