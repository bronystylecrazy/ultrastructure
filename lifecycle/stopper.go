package lifecycle

import (
	"sort"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func AppendStoppers(lc fx.Lifecycle, stoppers ...Stopper) {
	for _, stopper := range sortStoppersByPriority(stoppers) {
		lc.Append(fx.Hook{
			OnStop: stopper.Stop,
		})
	}
}

func sortStoppersByPriority(stoppers []Stopper) []Stopper {
	if len(stoppers) <= 1 {
		return stoppers
	}
	type keyedStopper struct {
		value    Stopper
		priority int
	}
	keyed := make([]keyedStopper, len(stoppers))
	for i, stopper := range stoppers {
		keyed[i] = keyedStopper{
			value:    stopper,
			priority: stopPriorityIndex(stopper),
		}
	}
	sort.SliceStable(keyed, func(i, j int) bool {
		return keyed[i].priority < keyed[j].priority
	})
	out := make([]Stopper, len(keyed))
	for i := range keyed {
		out[i] = keyed[i].value
	}
	return out
}

func stopPriorityIndex(stopper Stopper) int {
	values := di.FindAllMetadata[stopPriority](stopper)
	if len(values) == 0 {
		return int(Normal)
	}
	return values[len(values)-1].Index
}
