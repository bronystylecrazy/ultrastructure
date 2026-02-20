package database

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func Module(opts ...di.Node) di.Node {
	return di.Module(
		"us/database",
		di.Config[Config]("db"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
		di.Provide(NewDialector),
		di.Provide(NewGormDB),
		di.Provide(NewGormChecker),
		di.Options(di.ConvertAnys(opts)...),
	)
}
