package lifecycle

import (
	"go.uber.org/fx"
)

type StopParams struct {
	fx.In

	Lc       fx.Lifecycle
	Stoppers []Stopper `group:"auto-stoppers"`
}

func RegisterStoppers(params StopParams) {
	for _, stopper := range params.Stoppers {
		params.Lc.Append(fx.Hook{
			OnStop: stopper.Stop,
		})
	}
}
