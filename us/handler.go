package us

import (
	"reflect"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

var GroupHandler = "handlers"

type Handler interface {
	Handle(r fiber.Router)
}

func AsHandlerGroup(group ...string) ProvideOption {
	name := GroupHandler
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return provideOptionFunc(func(cfg *provideConfig) {
		cfg.exports = append(cfg.exports, exportSpec{
			typ:     reflect.TypeOf((*Handler)(nil)).Elem(),
			group:   name,
			grouped: true,
		})
	})
}

// AsHandler exposes a single Handler (non-grouped).
func AsHandler(name ...string) ProvideOption {
	return AsType[Handler](name...)
}

// InHandlers tags a Handler slice parameter for Invoke.
func InHandlers(group ...string) InvokeOption {
	name := GroupHandler
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return InTag(`group:"` + name + `"`)
}

// InHandlersTag returns a struct tag string for use with fx.ParamTags.
func InHandlersTag(group ...string) string {
	name := GroupHandler
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return InGroupTag(name)
}

// AsHandlerAnn returns fx.Annotations for providing Handler into a group via fx.Annotate.
func AsHandlerAnn(group ...string) []fx.Annotation {
	name := GroupHandler
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return AsGroupAnn[Handler](name)
}

// DecorateHandlers tags a Handler slice parameter for Decorate.
func DecorateHandlers(group ...string) DecorateOption {
	name := GroupHandler
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return DecorateTag(`group:"` + name + `"`)
}
