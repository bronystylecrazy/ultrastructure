package di

import "reflect"

type tagSet struct {
	// name/group identify the tagged export; typ is the concrete type.
	name  string
	group string
	typ   reflect.Type
}

type decorateEntry struct {
	// dec is the decorator; tagSets are the targets for this decorator.
	dec      decorateNode
	tagSets  []tagSet
	position int
}

type provideItem struct {
	// provideItem tracks a provider and its exported tag sets.
	pos     int
	node    Node
	tagSets []tagSet
}
