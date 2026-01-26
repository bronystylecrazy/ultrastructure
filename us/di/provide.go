package di

import (
	"fmt"
	"reflect"

	"github.com/bronystylecrazy/ultrastructure/us"
	"go.uber.org/fx"
)

// Provide declares a constructor and options; use App(...).Build() to compile.
func Provide(constructor any, opts ...any) Node {
	return provideNode{constructor: constructor, opts: opts}
}

// Supply declares a value and options; use App(...).Build() to compile.
func Supply(value any, opts ...any) Node {
	return supplyNode{value: value, opts: opts}
}

func constructorResultType(constructor any) (reflect.Type, error) {
	if constructor == nil {
		return nil, fmt.Errorf("constructor must not be nil")
	}
	fn := reflect.TypeOf(constructor)
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("constructor must be a function")
	}
	numOut := fn.NumOut()
	if numOut < 1 || numOut > 2 {
		return nil, fmt.Errorf("constructor must return 1 value (and optional error)")
	}
	if numOut == 2 && fn.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return nil, fmt.Errorf("constructor's second result must be error")
	}
	return fn.Out(0), nil
}

func buildProvideOptions(cfg bindConfig, constructor any, value any) ([]us.ProvideOption, []tagSet, error) {
	var provideOpts []us.ProvideOption
	var tagSets []tagSet

	if cfg.includeSelf {
		provideOpts = append(provideOpts, us.AsSelf())
	}
	if cfg.privateSet {
		if cfg.privateValue {
			provideOpts = append(provideOpts, us.Private())
		} else {
			provideOpts = append(provideOpts, us.Public())
		}
	}

	var baseType reflect.Type
	getBaseType := func() (reflect.Type, error) {
		if baseType != nil {
			return baseType, nil
		}
		if constructor != nil {
			t, err := constructorResultType(constructor)
			if err != nil {
				return nil, err
			}
			baseType = t
			return baseType, nil
		}
		if value != nil {
			baseType = reflect.TypeOf(value)
			if baseType == nil {
				return nil, fmt.Errorf("value must not be nil")
			}
			return baseType, nil
		}
		return nil, fmt.Errorf("cannot infer type")
	}
	if cfg.pendingName != "" || cfg.pendingGroup != "" {
		if len(cfg.exports) > 0 {
			return nil, nil, fmt.Errorf("name/group must apply to a single As, not mixed")
		}
		if _, err := getBaseType(); err != nil {
			return nil, nil, err
		}
		if cfg.pendingName != "" {
			provideOpts = append(provideOpts, us.AsTypeOf(baseType, cfg.pendingName))
			tagSets = append(tagSets, tagSet{name: cfg.pendingName, typ: baseType})
		}
		if cfg.pendingGroup != "" {
			provideOpts = append(provideOpts, us.AsGroupOf(baseType, cfg.pendingGroup))
			tagSets = append(tagSets, tagSet{group: cfg.pendingGroup, typ: baseType})
		}
	}

	hasUntagged := false
	for _, exp := range cfg.exports {
		if exp.grouped && exp.named {
			return nil, nil, fmt.Errorf("export cannot be both grouped and named")
		}
		if exp.grouped {
			provideOpts = append(provideOpts, us.AsGroupOf(exp.typ, exp.group))
			tagSets = append(tagSets, tagSet{group: exp.group, typ: exp.typ})
			continue
		}
		if exp.named {
			provideOpts = append(provideOpts, us.AsTypeOf(exp.typ, exp.name))
			tagSets = append(tagSets, tagSet{name: exp.name, typ: exp.typ})
			continue
		}
		provideOpts = append(provideOpts, us.AsTypeOf(exp.typ))
		tagSets = append(tagSets, tagSet{typ: exp.typ})
		hasUntagged = true
	}

	if cfg.includeSelf && !hasUntagged {
		if _, err := getBaseType(); err != nil {
			return nil, nil, err
		}
		tagSets = append(tagSets, tagSet{typ: baseType})
	}
	if len(tagSets) == 0 {
		if _, err := getBaseType(); err != nil {
			return nil, nil, err
		}
		tagSets = append(tagSets, tagSet{typ: baseType})
	}

	if len(cfg.autoGroups) > 0 && !cfg.ignoreAuto {
		if _, err := getBaseType(); err != nil {
			return nil, nil, err
		}
		for _, rule := range cfg.autoGroups {
			if rule.iface == nil {
				continue
			}
			if rule.filter != nil && !rule.filter(baseType) {
				continue
			}
			if !implementsInterface(baseType, rule.iface) {
				continue
			}
			exists := false
			for _, ts := range tagSets {
				if ts.group == rule.group && typesMatch(ts.typ, rule.iface) {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
			provideOpts = append(provideOpts, us.AsGroupOf(rule.iface, rule.group))
			tagSets = append(tagSets, tagSet{group: rule.group, typ: rule.iface})
			if rule.asSelf {
				provideOpts = append(provideOpts, us.AsSelf())
				tagSets = append(tagSets, tagSet{typ: baseType})
			}
		}
	}

	return provideOpts, tagSets, nil
}

func implementsInterface(base reflect.Type, iface reflect.Type) bool {
	if iface == nil || iface.Kind() != reflect.Interface {
		return false
	}
	if base.Implements(iface) {
		return true
	}
	if base.Kind() != reflect.Pointer {
		if reflect.PointerTo(base).Implements(iface) {
			return true
		}
	}
	return false
}

type provideNode struct {
	constructor any
	opts        []any
}

func (n provideNode) Build() (fx.Option, error) {
	cfg, _, extra, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, err
	}
	provideOpts, _, err := buildProvideOptions(cfg, n.constructor, nil)
	if err != nil {
		return nil, err
	}
	provideOpt := us.Provide(n.constructor, provideOpts...)
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	if len(out) == 1 {
		return out[0], nil
	}
	return fx.Options(out...), nil
}

type supplyNode struct {
	value any
	opts  []any
}

func (n supplyNode) Build() (fx.Option, error) {
	cfg, _, extra, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, err
	}
	provideOpts, _, err := buildProvideOptions(cfg, nil, n.value)
	if err != nil {
		return nil, err
	}
	args := make([]any, 0, 1+len(provideOpts))
	args = append(args, n.value)
	for _, opt := range provideOpts {
		args = append(args, opt)
	}
	provideOpt := us.Supply(args...)
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	if len(out) == 1 {
		return out[0], nil
	}
	return fx.Options(out...), nil
}
