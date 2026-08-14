package di

import "reflect"

// ProvidedTypes is implemented by nodes that supply values di did not build
// itself, such as configuration loaded straight into fx. Declaring the types
// puts those values within reach of Replace and Default, which otherwise cannot
// see past the node.
//
// Only untagged exports are addressable this way, which is what configuration
// and similar whole-value bindings use.
type ProvidedTypes interface {
	ProvidedTypes() []reflect.Type
}

func tagSetsForProvidedTypes(node ProvidedTypes) []tagSet {
	types := node.ProvidedTypes()
	if len(types) == 0 {
		return nil
	}
	out := make([]tagSet, 0, len(types))
	for _, typ := range types {
		if typ == nil {
			continue
		}
		out = append(out, tagSet{typ: typ})
	}
	return out
}
