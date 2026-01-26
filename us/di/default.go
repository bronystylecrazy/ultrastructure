package di

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/us"
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
		return nil, fmt.Errorf("default does not support decorate options")
	}
	if cfg.privateSet {
		return nil, fmt.Errorf("default does not support private/public")
	}
	if cfg.pendingName != "" || cfg.pendingGroup != "" {
		return nil, fmt.Errorf("default does not support named or grouped exports")
	}
	var provideOpts []us.ProvideOption
	if cfg.includeSelf {
		provideOpts = append(provideOpts, us.AsSelf())
	}
	for _, exp := range cfg.exports {
		if exp.grouped {
			return nil, fmt.Errorf("default does not support groups")
		}
		if exp.named {
			return nil, fmt.Errorf("default does not support named exports")
		}
		provideOpts = append(provideOpts, us.AsTypeOf(exp.typ))
	}
	if len(provideOpts) == 0 {
		return us.Supply(n.value), nil
	}
	args := make([]any, 0, 1+len(provideOpts))
	args = append(args, n.value)
	for _, opt := range provideOpts {
		args = append(args, opt)
	}
	return us.Supply(args...), nil
}
