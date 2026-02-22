package lc

import (
	"sort"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func AppendStarters(lc fx.Lifecycle, starters ...Starter) {
	for _, starter := range sortStartersByPriority(starters) {
		lc.Append(fx.Hook{
			OnStart: starter.Start,
		})
	}
}

func sortStartersByPriority(starters []Starter) []Starter {
	if len(starters) <= 1 {
		return starters
	}
	type keyedStarter struct {
		value    Starter
		priority int64
	}
	keyed := make([]keyedStarter, len(starters))
	for i, starter := range starters {
		keyed[i] = keyedStarter{
			value:    starter,
			priority: startPriorityIndex(starter),
		}
	}
	sort.SliceStable(keyed, func(i, j int) bool {
		return keyed[i].priority < keyed[j].priority
	})
	out := make([]Starter, len(keyed))
	for i := range keyed {
		out[i] = keyed[i].value
	}
	return out
}

func startPriorityIndex(starter Starter) int64 {
	values := di.FindAllMetadata[startPriority](starter)
	if len(values) == 0 {
		return int64(Normal)
	}
	return values[len(values)-1].Index
}
