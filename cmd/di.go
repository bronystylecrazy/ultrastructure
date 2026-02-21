package cmd

import (
	"github.com/bronystylecrazy/ultrastructure/di"
)

func UseServiceCommands() di.Node {
	return di.Options(
		di.Provide(NewServiceCommand),
		OnRun("service", UseServiceController()),
	)
}

func UseBasicCommands() di.Node {
	return di.Options(
		di.Provide(NewHealthcheckCommand),
		di.Provide(NewHelpCommand),
		di.Provide(NewVersionCommand),
	)
}
