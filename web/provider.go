package web

import (
	"math"

	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
)

var HandlersGroupName = "us.handlers"

var OtelScope = "web.http"

func Provide(extends ...di.Node) di.Node {

	nodes := []di.Node{
		di.AutoGroup[FiberConfigurer](FiberConfigurersGroupName),

		cfg.Config[Config]("web", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),

		cfg.Config[FiberConfig]("web.fiber", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),

		di.Provide(NewRegistryContainer),
		di.Provide(NewRegistryLifecycle),
		di.Provide(NewModuleRouter),

		di.Provide(
			NewOtelMiddleware,
			Priority(math.MinInt32), otel.Layer(OtelScope),
		),
		di.Provide(
			NewFiberServer,
			di.VariadicGroup(FiberConfigurersGroupName),
			di.AsSelf[Server](),
		),
	}

	nodes = append(nodes, extends...)

	return di.Options(di.ConvertAnys(nodes)...)
}
