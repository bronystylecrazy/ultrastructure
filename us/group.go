package us

import (
	"reflect"

	"go.uber.org/fx"
)

// Group returns a reusable helper for providing and invoking grouped values.
func Group[T any](name string) GroupSpec[T] {
	return GroupSpec[T]{
		name: name,
		typ:  reflect.TypeOf((*T)(nil)).Elem(),
	}
}

type GroupSpec[T any] struct {
	name string
	typ  reflect.Type
}

func (g GroupSpec[T]) As() ProvideOption {
	return asGroupSpec(g.name, g.typ)
}

func (g GroupSpec[T]) In() InvokeOption {
	return InGroup(g.name)
}

// AsGroup exposes constructor results as type T in a group.
func AsGroup[T any](name string) ProvideOption {
	return asGroupSpec(name, reflect.TypeOf((*T)(nil)).Elem())
}

// AsGroupOf exposes constructor results as a type into a group using reflection.
func AsGroupOf(typ reflect.Type, name string) ProvideOption {
	return asGroupSpec(name, typ)
}

// AsType exposes constructor results as type T without a group.
func AsType[T any](name ...string) ProvideOption {
	n := ""
	if len(name) > 0 && name[0] != "" {
		n = name[0]
	}
	return asTypeSpec(reflect.TypeOf((*T)(nil)).Elem(), n)
}

// AsTypeOf exposes constructor results as a type without a group using reflection.
func AsTypeOf(typ reflect.Type, name ...string) ProvideOption {
	n := ""
	if len(name) > 0 && name[0] != "" {
		n = name[0]
	}
	return asTypeSpec(typ, n)
}

// InGroup tags the next Invoke parameter to receive a grouped value.
func InGroup(name string) InvokeOption {
	return InTag(`group:"` + name + `"`)
}

// DecorateGroup tags the next Decorate parameter to receive a grouped value.
func DecorateGroup(name string) DecorateOption {
	return DecorateTag(`group:"` + name + `"`)
}

// InGroupTag returns a struct tag string for use with fx.ParamTags.
func InGroupTag(name string) string {
	return `group:"` + name + `"`
}

// OptionalTag returns a struct tag string for use with fx.ParamTags.
func OptionalTag() string {
	return `optional:"true"`
}

// NameTag returns a struct tag string for use with fx.ParamTags.
func NameTag(name string) string {
	return `name:"` + name + `"`
}

// AsGroupAnn returns fx.Annotations for providing T into a group via fx.Annotate.
func AsGroupAnn[T any](name string) []fx.Annotation {
	return []fx.Annotation{
		fx.As(new(T)),
		fx.ResultTags(`group:"` + name + `"`),
	}
}

func asGroupSpec(name string, typ reflect.Type) ProvideOption {
	return provideOptionFunc(func(cfg *provideConfig) {
		cfg.exports = append(cfg.exports, exportSpec{
			typ:     typ,
			group:   name,
			grouped: true,
		})
	})
}

func asTypeSpec(typ reflect.Type, name string) ProvideOption {
	return provideOptionFunc(func(cfg *provideConfig) {
		exp := exportSpec{
			typ:     typ,
			group:   "",
			grouped: false,
		}
		if name != "" {
			exp.name = name
			exp.named = true
		}
		cfg.exports = append(cfg.exports, exp)
	})
}
