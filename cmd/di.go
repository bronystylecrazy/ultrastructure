package cmd

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func UseServiceCommands() di.Node {
	return di.Options(
		Use("service",
			UseServiceController(),
			di.Provide(NewServiceCommand),
		),
	)
}

func UseBasicCommands() di.Node {
	return di.Options(
		Use("healthcheck", di.Provide(NewHealthcheckCommand)),
		Use("help", di.Provide(NewHelpCommand)),
		Use("version", di.Provide(NewVersionCommand)),
	)
}
