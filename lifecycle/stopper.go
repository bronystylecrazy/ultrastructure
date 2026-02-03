package lifecycle

import (
	"go.uber.org/fx"
)

func AppendStoppers(lc fx.Lifecycle, stoppers ...Stopper) {
	for _, stopper := range stoppers {
		lc.Append(fx.Hook{
			OnStop: stopper.Stop,
		})
	}
}
