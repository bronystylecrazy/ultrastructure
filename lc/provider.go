package lc

import (
	"context"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

var StartersGroupName = "lc.starters"
var StoppersGroupName = "lc.stoppers"

func NewBackgroundContext(lc fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
	return ctx
}

func Providers() di.Node {
	return di.Options(
		di.AutoGroup[Starter](StartersGroupName),
		di.AutoGroup[Stopper](StoppersGroupName),
		di.Provide(NewBackgroundContext),
		di.Invoke(AppendStarters, di.Params(``, di.Group(StartersGroupName))),
		di.Invoke(AppendStoppers, di.Params(``, di.Group(StoppersGroupName))),
	)
}
