package lc

import "github.com/bronystylecrazy/ultrastructure/di"

type PriorityLevel = di.PriorityLevel

const (
	Earliest = di.Earliest
	Earlier  = di.Earlier
	Normal   = di.Normal
	Later    = di.Later
	Latest   = di.Latest
)

type startPriority struct {
	Index int64
}

type stopPriority struct {
	Index int64
}

// StartPriority controls lifecycle starter registration order.
// Lower values register earlier. Stable order is preserved for ties.
func StartPriority(value PriorityLevel) di.Option {
	return di.Metadata(startPriority{Index: int64(value)})
}

// StopPriority controls lifecycle stopper registration order.
// Lower values register earlier. Stable order is preserved for ties.
func StopPriority(value PriorityLevel) di.Option {
	return di.Metadata(stopPriority{Index: int64(value)})
}
