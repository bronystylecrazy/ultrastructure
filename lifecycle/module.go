package lifecycle

import (
	"context"

	"github.com/bronystylecrazy/ultrastructure/di"
)

var StartersGroupName = "auto-starters"
var StoppersGroupName = "auto-stoppers"

func NewBackgroundContext() context.Context {
	return context.Background()
}

func Module() di.Node {
	return di.Options(
		di.AutoGroup[Starter](StartersGroupName),
		di.AutoGroup[Stopper](StoppersGroupName),
		di.Provide(NewBackgroundContext),
		di.Invoke(RegisterStarters),
		di.Invoke(RegisterStoppers),
	)
}
