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

func Priority(value PriorityLevel) di.Option {
	return di.Priority(value)
}

func Between(lower, upper PriorityLevel) PriorityLevel {
	return di.Between(lower, upper)
}

func resolvePriority(handler Handler) int {
	priority := int(Normal)
	if p, ok := di.PriorityIndex(handler); ok {
		priority = p
	} else if meta, ok := di.ReflectMetadata[[]any](handler); ok {
		for _, item := range meta {
			switch v := item.(type) {
			case PriorityLevel:
				priority = int(v)
			case int:
				priority = v
			}
		}
	}
	order := 0
	if o, ok := di.OrderIndex(handler); ok {
		order = o
	}
	return priority*1_000_000 + order
}
