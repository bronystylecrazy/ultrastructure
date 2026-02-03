package otel

import "github.com/bronystylecrazy/ultrastructure/di"

// LayerMetadata describes a logical layer used for tracer naming.
type LayerMetadata struct {
	Kind string
	Name string
}

// Layer packs layer metadata into a single DI option.
func Layer(name string) di.Option {
	return di.Metadata(LayerMetadata{Kind: "layer", Name: name})
}
