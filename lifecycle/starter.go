package lifecycle

import (
	"go.uber.org/fx"
)

func AppendStarters(lc fx.Lifecycle, starters ...Starter) {
	for _, starter := range starters {
		lc.Append(fx.Hook{
			OnStart: starter.Start,
		})
	}
}
