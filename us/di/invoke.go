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
		return fx.Invoke(fx.Annotate(n.function, fx.ParamTags(n.paramTagsOverride...))), nil
	}
	var cfg paramConfig
	for _, opt := range n.opts {
		if opt != nil {
			opt.applyParam(&cfg)
		}
		if cfg.err != nil {
			return nil, cfg.err
		}
	}
	if len(cfg.tags) == 0 {
		return fx.Invoke(n.function), nil
	}
	return fx.Invoke(fx.Annotate(n.function, fx.ParamTags(cfg.tags...))), nil
}
