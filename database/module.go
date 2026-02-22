package database

import (
	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
)

func Module(opts ...di.Node) di.Node {
	return di.Module(
		"us/database",
		cfg.Config[Config]("db", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
		di.Options(di.ConvertAnys(opts)...),
	)
}
