package di

import (
	"fmt"

	"go.uber.org/fx"
)

// Populate declares a populate node.
func Populate(args ...any) Node {
	var targets []any
	var opts []Option
	for _, arg := range args {
		if arg == nil {
			continue
		}
		if opt, ok := arg.(Option); ok {
			opts = append(opts, opt)
			continue
		}
		targets = append(targets, arg)
	}
	return populateNode{targets: targets, opts: opts}
}

type populateNode struct {
	targets []any
	opts    []Option
}

func (n populateNode) Build() (fx.Option, error) {
	if len(n.targets) == 0 {
		return fx.Populate(), nil
	}
	if len(n.opts) == 0 {
		return fx.Populate(n.targets...), nil
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
		return fx.Populate(n.targets...), nil
	}
	if len(n.targets) != 1 {
		return nil, fmt.Errorf("Populate with tags requires a single target")
	}
	annotated := fx.Annotate(n.targets[0], fx.ParamTags(cfg.tags...))
	return fx.Populate(annotated), nil
}
