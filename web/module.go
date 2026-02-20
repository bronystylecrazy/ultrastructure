package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
)

var HandlersGroupName = "us.handlers"

func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.Module(
			"us.web",
			di.AutoGroup[FiberConfigConfigurer](FiberConfigConfigurersGroupName),
			di.Config[Config]("web"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),

			di.Config[FiberConfig]("web.fiber"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
			di.Provide(NewRegistryContainer),
			di.Provide(NewRegistryLifecycle),
			di.Provide(NewModuleRouter),
			di.Provide(
				NewOtelMiddleware,
				Priority(Earliest), IgnoreAutoGroupHandlers(),
				otel.Layer("web.http"),
			),
			di.Provide(NewFiberApp, di.Params(nil, di.Group(FiberConfigConfigurersGroupName))),
			di.Options(di.ConvertAnys(extends)...),
		),
	)
}
