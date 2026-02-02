package database

import "github.com/bronystylecrazy/ultrastructure/di"

func Module(opts ...di.Node) di.Node {
	return di.Module(
		"us/database",
		di.Config[Config]("db"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),
		di.Provide(NewPostgresDialector),
		di.Provide(NewGormDB),
		di.Options(di.ConvertAnys(opts)...),
	)
}
