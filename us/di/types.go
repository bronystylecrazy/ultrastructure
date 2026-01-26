package di

import "reflect"

type tagSet struct {
	name  string
	group string
	typ   reflect.Type
}

type decorateEntry struct {
	dec      decorateNode
	tagSets  []tagSet
	position int
}

type provideItem struct {
	pos     int
	node    Node
	tagSets []tagSet
}
