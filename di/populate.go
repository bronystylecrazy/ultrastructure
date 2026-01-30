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
			// Collect param options separately from targets.
			opts = append(opts, opt)
			continue
		}
		// Non-option args are populate targets.
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
		// No targets: defer to fx.Populate() with no args.
		return fx.Populate(), nil
	}
	if len(n.opts) == 0 {
		// No tags: simple populate.
		return fx.Populate(n.targets...), nil
	}
	var cfg paramConfig
	if err := applyParamOptions(n.opts, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.tags) == 0 {
		// Tags were not provided; populate directly.
		return fx.Populate(n.targets...), nil
	}
	if len(n.targets) != 1 {
		// Fx only supports tags when a single target is provided.
		return nil, fmt.Errorf(errParamTagsSingleTarget)
	}
	annotated := fx.Annotate(n.targets[0], fx.ParamTags(cfg.tags...))
	return fx.Populate(annotated), nil
}
