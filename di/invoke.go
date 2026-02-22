package di

import (
	"reflect"
	"runtime"

	"go.uber.org/fx"
)

// Invoke declares an invoke node.
func Invoke(function any, opts ...Option) Node {
	file, line := invokeCallSite()
	return invokeNode{function: function, opts: opts, sourceFile: file, sourceLine: line}
}

type invokeNode struct {
	function          any
	opts              []Option
	sourceFile        string
	sourceLine        int
	paramTagsOverride []string
}

func (n invokeNode) Build() (fx.Option, error) {
	var cfg paramConfig
	if err := applyParamOptions(n.opts, &cfg); err != nil {
		return nil, err
	}
	expected := 0
	if fnType := reflect.TypeOf(n.function); fnType != nil && fnType.Kind() == reflect.Func {
		expected = fnType.NumIn()
	}
	if err := resolveVariadicParamTags(n.function, &cfg, expected); err != nil {
		return nil, err
	}
	if err := validateParamCountForFunction(n.function, cfg, n.sourceFile, n.sourceLine, errInvokeParamsCountMismatch, "invoke signature"); err != nil {
		return nil, err
	}
	if n.paramTagsOverride != nil {
		if hasAnyTag(n.paramTagsOverride) {
			// Use rewritten tags when replacements modified param tags.
			return fx.Invoke(fx.Annotate(n.function, fx.ParamTags(n.paramTagsOverride...))), nil
		}
		return fx.Invoke(n.function), nil
	}
	if len(cfg.tags) == 0 {
		// No param tags: invoke directly.
		return fx.Invoke(n.function), nil
	}
	// Apply positional param tags to the invoke function.
	return fx.Invoke(fx.Annotate(n.function, fx.ParamTags(cfg.tags...))), nil
}

func invokeCallSite() (string, int) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "", 0
	}
	return file, line
}
