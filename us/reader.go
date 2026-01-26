package us

import (
	"reflect"

	"go.uber.org/fx"
)

var GroupReader = "readers"

type Reader interface {
	Read() string
}

func AsReaderGroup(group ...string) ProvideOption {
	name := GroupReader
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return provideOptionFunc(func(cfg *provideConfig) {
		cfg.exports = append(cfg.exports, exportSpec{
			typ:     reflect.TypeOf((*Reader)(nil)).Elem(),
			group:   name,
			grouped: true,
		})
	})
}

// AsReader exposes a single Reader (non-grouped).
func AsReader(name ...string) ProvideOption {
	return AsType[Reader](name...)
}

// InReaders tags a Reader slice parameter for Invoke.
func InReaders(group ...string) InvokeOption {
	name := GroupReader
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return InTag(`group:"` + name + `"`)
}

// InReadersTag returns a struct tag string for use with fx.ParamTags.
func InReadersTag(group ...string) string {
	name := GroupReader
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return InGroupTag(name)
}

// AsReaderAnn returns fx.Annotations for providing Reader into a group via fx.Annotate.
func AsReaderAnn(group ...string) []fx.Annotation {
	name := GroupReader
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return AsGroupAnn[Reader](name)
}

// DecorateReaders tags a Reader slice parameter for Decorate.
func DecorateReaders(group ...string) DecorateOption {
	name := GroupReader
	if len(group) > 0 && group[0] != "" {
		name = group[0]
	}
	return DecorateTag(`group:"` + name + `"`)
}
