package us

import "go.uber.org/fx"

var WebModuleName = "infra/web"

func WebModule(options ...fx.Option) fx.Option {
	return fx.Module(
		WebModuleName,
		fx.Options(options...),
	)
}
