package web

import "github.com/bronystylecrazy/ultrastructure/di"

type PriorityLevel int

const (
	Earliest PriorityLevel = -200
	Earlier  PriorityLevel = -100
	Normal   PriorityLevel = 0
	Later    PriorityLevel = 100
	Latest   PriorityLevel = 200
)

func Priority(value PriorityLevel) di.Option {
	return di.Metadata(value)
}

func Between(lower, upper PriorityLevel) PriorityLevel {
	if lower == upper {
		return lower
	}
	if lower > upper {
		lower, upper = upper, lower
	}
	return PriorityLevel(int(lower) + (int(upper)-int(lower))/2)
}

func resolvePriority(handler Handler) PriorityLevel {
	meta, ok := di.ReflectMetadata[[]any](handler)
	if !ok {
		return Normal
	}
	priority := Normal
	for _, item := range meta {
		switch v := item.(type) {
		case PriorityLevel:
			priority = v
		case int:
			priority = PriorityLevel(v)
		}
	}
	return priority
}
