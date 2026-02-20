package cmd

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func UseServiceCommands() di.Node {
	return di.Options(
		OnRun("service",
			UseServiceController(),
			di.Provide(NewServiceCommand),
		),
	)
}

func UseBasicCommands() di.Node {
	return di.Options(
		OnRun("healthcheck", di.Provide(NewHealthcheckCommand)),
		OnRun("help", di.Provide(NewHelpCommand)),
		OnRun("version", di.Provide(NewVersionCommand)),
	)
}
