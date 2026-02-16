package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/gofiber/fiber/v3"
)

var HandlersGroupName = "us.handlers"

func Module(extends ...di.Node) di.Node {
	return di.Options(
		// auto discovery for handlers
		di.AutoGroup[Handler](HandlersGroupName),
		di.Module(
			"us.web",

			di.Config[Config]("web"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),

			di.Config[FiberConfig]("web.fiber"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
			di.Provide(NewFiberApp, di.AsSelf[fiber.Router]()),
			di.Provide(NewOtelMiddleware, otel.Layer("web.http"), IgnoreAutoGroupHandlers(), Priority(Earliest)),
			di.Options(di.ConvertAnys(extends)...),
		),
	)
}
