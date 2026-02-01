package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func Module(options ...di.Node) di.Node {
	return di.Module(
		"infrastructure/web",
		di.Config[Config]("web",
			di.ConfigDefault("web.host", "0.0.0.0"),
			di.ConfigDefault("web.port", "8080"),
		),
		di.Provide(NewFiberApp),
		di.Options(options),
	)
}
