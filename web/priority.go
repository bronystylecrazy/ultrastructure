package web

import "github.com/bronystylecrazy/ultrastructure/di"

type PriorityLevel = di.PriorityLevel

const (
	Earliest = di.Earliest
	Earlier  = di.Earlier
	Normal   = di.Normal
	Later    = di.Later
	Latest   = di.Latest
)

type webPriority struct {
	Index int
}

func Priority(value PriorityLevel) di.Option {
	return di.Metadata(webPriority{Index: int(value)})
}

func Between(lower, upper PriorityLevel) PriorityLevel {
	return di.Between(lower, upper)
}

func resolvePriority(handler Handler) int {
	priority := int(Normal)
	values := di.FindAllMetadata[webPriority](handler)
	if len(values) > 0 {
		priority = values[len(values)-1].Index
	}
	order := 0
	if o, ok := di.OrderIndex(handler); ok {
		order = o
	}
	return priority*1_000_000 + order
}
