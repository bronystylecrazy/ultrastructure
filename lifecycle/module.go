package lifecycle

import (
	"context"

	"github.com/bronystylecrazy/ultrastructure/di"
)

var StartersGroupName = "lifecycle.starters"
var StoppersGroupName = "lifecycle.stoppers"

func NewBackgroundContext() context.Context {
	return context.Background()
}

func Module() di.Node {
	return di.Options(
		di.AutoGroup[Starter](StartersGroupName),
		di.AutoGroup[Stopper](StoppersGroupName),
		di.Provide(NewBackgroundContext),
		di.Invoke(AppendStarters, di.Params(``, di.Group(StartersGroupName))),
		di.Invoke(AppendStoppers, di.Params(``, di.Group(StoppersGroupName))),
	)
}
