package web

// SwaggerCustomizersGroupName is the DI group used for SwaggerCustomizer implementations.
const SwaggerCustomizersGroupName = "web.swagger.customizers"
const SwaggerPreCustomizersGroupName = "web.swagger.customizers.pre"
const SwaggerPostCustomizersGroupName = "web.swagger.customizers.post"

// SwaggerCustomizer allows DI-provided structs to mutate route metadata during autoswagger generation.
type SwaggerCustomizer interface {
	CustomizeSwagger(ctx *SwaggerContext)
}

// SwaggerPreRun is an optional interface for logic that should run
// before the main customization phase.
type SwaggerPreRun interface {
	PreCustomizeSwagger(ctx *SwaggerContext)
}

// SwaggerPostRun is an optional interface for logic that should run
// after operation generation for the current route.
type SwaggerPostRun interface {
	PostCustomizeSwagger(ctx *SwaggerContext)
}

// SwaggerCustomizeFunc adapts a function to SwaggerCustomizer.
type SwaggerCustomizeFunc func(ctx *SwaggerContext)

func (f SwaggerCustomizeFunc) CustomizeSwagger(ctx *SwaggerContext) {
	if f != nil {
		f(ctx)
	}
}
