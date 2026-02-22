package database

import (
	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
)

func Providers(opts ...di.Node) di.Node {
	nodes := []any{
		cfg.Config[Config]("db", cfg.WithSourceFile("config.toml"), cfg.WithType("toml")),
	}
	nodes = append(nodes, di.ConvertAnys(opts)...)
	return di.Options(nodes...)
}
