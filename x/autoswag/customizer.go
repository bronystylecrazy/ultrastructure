package autoswag

const CustomizersGroupName = "autoswag.customizers"
const PreCustomizersGroupName = "autoswag.customizers.pre"
const PostCustomizersGroupName = "autoswag.customizers.post"

type Customizer interface {
	CustomizeSwagger(ctx *Context)
}

type PreRun interface {
	PreCustomizeSwagger(ctx *Context)
}

type PostRun interface {
	PostCustomizeSwagger(ctx *Context)
}

type CustomizeFunc func(ctx *Context)

func (f CustomizeFunc) CustomizeSwagger(ctx *Context) {
	if f != nil {
		f(ctx)
	}
}
