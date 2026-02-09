package database

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func Module(opts ...di.Node) di.Node {
	return di.Module(
		"us/database",
		di.Config[Config]("db"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
		di.Provide(NewPostgresDialector),
		di.Provide(NewGormDB),
		di.Provide(NewGormOtel),
		di.Invoke(GormCheck),
		di.Options(di.ConvertAnys(opts)...),
	)
}
