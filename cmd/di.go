package cmd

import "github.com/bronystylecrazy/ultrastructure/di"

func UseBasicCommands() di.Node {
	return di.Options(
		Use("help", di.Provide(NewHelpCommand)),
		Use("version", di.Provide(NewVersionCommand)),
	)
}
