package log

import (
	"github.com/bronystylecrazy/ultrastructure/build"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

var ModuleName = "ultrastructure/log"

func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.Provide(NewZapLogger),
		fx.WithLogger(NewEventLogger),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),
		di.Config[Config]("log",
			di.Switch(
				di.Case(build.IsDevelopment(), di.ConfigDefault("log.level", "debug")),
				di.Case(build.IsProduction(), di.ConfigDefault("log.level", "info")),
				di.DefaultCase(di.ConfigDefault("log.level", "info")),
			),
		),
		di.Options(di.ConvertAnys(extends)...),
	)
}
