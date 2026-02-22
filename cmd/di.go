package cmd

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lc"
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
		di.Provide(NewVersionCommand),
	)
}

func UseServiceController() di.Node {
	return di.Provide(
		NewServiceController,
		di.As[ServiceController](),
		di.AutoGroupIgnoreType[lc.Starter](),
		di.AutoGroupIgnoreType[lc.Stopper](),
	)
}

func UseServiceRuntime() di.Node {
	return di.Provide(NewServiceRuntimeHook, di.As[PreRunner]())
}
