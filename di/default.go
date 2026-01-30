package di

import (
	"fmt"

	"go.uber.org/fx"
)

// Default declares a binding used only if no matching Provide/Supply exists.
func Default(value any, opts ...any) Node {
	return defaultNode{value: value, opts: opts}
}

type defaultNode struct {
	value any
	opts  []any
}

func (n defaultNode) Build() (fx.Option, error) {
	cfg, decorators, _, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, err
	}
	if len(decorators) > 0 {
		return nil, fmt.Errorf(errDefaultNoDecorate)
	}
	if cfg.privateSet {
		return nil, fmt.Errorf(errDefaultNoPrivatePublic)
	}
	if cfg.pendingName != "" || cfg.pendingGroup != "" {
		return nil, fmt.Errorf(errDefaultNoNamedOrGroupedExports)
	}
	// Defaults are only allowed for ungrouped, unnamed exports.
	for _, exp := range cfg.exports {
		if exp.grouped {
			return nil, fmt.Errorf(errDefaultNoGroups)
		}
		if exp.named {
			return nil, fmt.Errorf(errDefaultNoNamedExports)
		}
	}
	spec, _, err := buildProvideSpec(cfg, nil, n.value)
	if err != nil {
		return nil, err
	}
	return buildProvideSupplyOption(spec, n.value)
}
