package di

// PriorityLevel controls auto-group ordering for provided values.
// Lower values are ordered earlier.
type PriorityLevel int64

type priorityOrder struct {
	Index int64
}

const (
	Earliest PriorityLevel = -10000
	Earlier  PriorityLevel = -5000
	Normal   PriorityLevel = 0
	Later    PriorityLevel = 5000
	Latest   PriorityLevel = 10000
)

// Priority overrides the auto-group order for a provided value.
func Priority(value PriorityLevel) Option {
	return Metadata(priorityOrder{Index: int64(value)})
}

// PriorityIndex returns the priority order index if present.
func PriorityIndex(value any) (int64, bool) {
	raw, ok := ReflectMetadataAny(value)
	if !ok {
		return 0, false
	}
	meta, ok := raw.([]any)
	if !ok {
		return 0, false
	}
	found := false
	priority := int64(0)
	for _, item := range meta {
		if ord, ok := item.(priorityOrder); ok {
			priority = ord.Index
			found = true
		}
	}
	return priority, found
}

// OrderIndex returns the auto-group order index if present.
func OrderIndex(value any) (int, bool) {
	raw, ok := ReflectMetadataAny(value)
	if !ok {
		return 0, false
	}
	meta, ok := raw.([]any)
	if !ok {
		return 0, false
	}
	found := false
	order := 0
	for _, item := range meta {
		if ord, ok := item.(autoGroupOrder); ok {
			order = ord.Index
			found = true
		}
	}
	return order, found
}

// Between returns a PriorityLevel between lower and upper.
func Between(lower, upper PriorityLevel) PriorityLevel {
	if lower == upper {
		return lower
	}
	if lower > upper {
		lower, upper = upper, lower
	}
	return PriorityLevel(int64(lower) + (int64(upper)-int64(lower))/2)
}
