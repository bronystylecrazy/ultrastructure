package di

import "go.uber.org/fx"

// Invoke declares an invoke node.
func Invoke(function any, opts ...Option) Node {
	return invokeNode{function: function, opts: opts}
}

type invokeNode struct {
	function          any
	opts              []Option
	paramTagsOverride []string
}

func (n invokeNode) Build() (fx.Option, error) {
	if n.paramTagsOverride != nil {
		// Use rewritten tags when replacements modified param tags.
		return fx.Invoke(fx.Annotate(n.function, fx.ParamTags(n.paramTagsOverride...))), nil
	}
	var cfg paramConfig
	if err := applyParamOptions(n.opts, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.tags) == 0 {
		// No param tags: invoke directly.
		return fx.Invoke(n.function), nil
	}
	// Apply positional param tags to the invoke function.
	return fx.Invoke(fx.Annotate(n.function, fx.ParamTags(cfg.tags...))), nil
}
