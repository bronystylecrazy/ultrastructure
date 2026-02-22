package web

import (
	"sort"

	"github.com/bronystylecrazy/ultrastructure/di"
)

type PriorityLevel int64

const (
	Earliest PriorityLevel = PriorityLevel(di.Earliest)
	Earlier  PriorityLevel = PriorityLevel(di.Earlier)
	Normal   PriorityLevel = PriorityLevel(di.Normal)
	Later    PriorityLevel = PriorityLevel(di.Later)
	Latest   PriorityLevel = PriorityLevel(di.Latest)
)

type webPriority struct {
	Index int64
}

func Priority(value PriorityLevel) di.Option {
	return di.Metadata(webPriority{Index: int64(value)})
}

func Between(lower, upper PriorityLevel) PriorityLevel {
	return PriorityLevel(di.Between(di.PriorityLevel(lower), di.PriorityLevel(upper)))
}

func ResolvePriority(handler Handler) int64 {
	priority := int64(Normal)
	values := di.FindAllMetadata[webPriority](handler)
	if len(values) > 0 {
		priority = values[len(values)-1].Index
	}
	order := int64(0)
	if o, ok := di.OrderIndex(handler); ok {
		order = int64(o)
	}
	return priority*1_000_000 + order
}

// Prioritize returns a stably sorted copy of handlers by web priority and DI order index.
func Prioritize(handlers []Handler) []Handler {
	ordered := append([]Handler(nil), handlers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ResolvePriority(ordered[i]) < ResolvePriority(ordered[j])
	})
	return ordered
}
