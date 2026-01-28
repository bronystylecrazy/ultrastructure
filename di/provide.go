package di

import (
	"fmt"
	"reflect"

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

type provideSpec struct {
	exports      []exportSpec
	includeSelf  bool
	privateSet   bool
	privateValue bool
}

func buildProvideSpec(cfg bindConfig, constructor any, value any) (provideSpec, []tagSet, error) {
	spec := provideSpec{
		includeSelf:  cfg.includeSelf,
		privateSet:   cfg.privateSet,
		privateValue: cfg.privateValue,
	}
	var tagSets []tagSet

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
			return provideSpec{}, nil, fmt.Errorf("name/group must apply to a single As, not mixed")
		}
		if _, err := getBaseType(); err != nil {
			return provideSpec{}, nil, err
		}
		if cfg.pendingName != "" {
			spec.exports = append(spec.exports, exportSpec{
				typ:   baseType,
				name:  cfg.pendingName,
				named: true,
			})
			tagSets = append(tagSets, tagSet{name: cfg.pendingName, typ: baseType})
		}
		if cfg.pendingGroup != "" {
			spec.exports = append(spec.exports, exportSpec{
				typ:     baseType,
				group:   cfg.pendingGroup,
				grouped: true,
			})
			tagSets = append(tagSets, tagSet{group: cfg.pendingGroup, typ: baseType})
		}
	}

	hasUntagged := false
	noExplicitExports := len(cfg.exports) == 0 && cfg.pendingName == "" && cfg.pendingGroup == "" && !cfg.includeSelf
	for _, exp := range cfg.exports {
		if exp.grouped && exp.named {
			return provideSpec{}, nil, fmt.Errorf("export cannot be both grouped and named")
		}
		if exp.grouped {
			tagSets = append(tagSets, tagSet{group: exp.group, typ: exp.typ})
		} else if exp.named {
			tagSets = append(tagSets, tagSet{name: exp.name, typ: exp.typ})
		} else {
			tagSets = append(tagSets, tagSet{typ: exp.typ})
			hasUntagged = true
		}
		spec.exports = append(spec.exports, exp)
	}

	if cfg.includeSelf && !hasUntagged {
		if _, err := getBaseType(); err != nil {
			return provideSpec{}, nil, err
		}
		tagSets = append(tagSets, tagSet{typ: baseType})
	}
	if len(tagSets) == 0 {
		if _, err := getBaseType(); err != nil {
			return provideSpec{}, nil, err
		}
		tagSets = append(tagSets, tagSet{typ: baseType})
	}

	if len(cfg.autoGroups) > 0 && !cfg.ignoreAuto {
		if _, err := getBaseType(); err != nil {
			return provideSpec{}, nil, err
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
			if hasExport(spec.exports, rule.iface, rule.group) {
				continue
			}
			if noExplicitExports {
				spec.includeSelf = true
			}
			spec.exports = append(spec.exports, exportSpec{
				typ:     rule.iface,
				group:   rule.group,
				grouped: true,
			})
			tagSets = append(tagSets, tagSet{group: rule.group, typ: rule.iface})
			if rule.asSelf {
				spec.includeSelf = true
			}
		}
	}

	if spec.includeSelf && !hasUntagged && !tagSetHasType(tagSets, baseType) {
		if _, err := getBaseType(); err != nil {
			return provideSpec{}, nil, err
		}
		tagSets = append(tagSets, tagSet{typ: baseType})
	}

	return spec, tagSets, nil
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

func hasExport(exports []exportSpec, typ reflect.Type, group string) bool {
	for _, exp := range exports {
		if !exp.grouped {
			continue
		}
		if exp.group != group {
			continue
		}
		if typesMatch(exp.typ, typ) {
			return true
		}
	}
	return false
}

func tagSetHasType(tagSets []tagSet, typ reflect.Type) bool {
	if typ == nil {
		return false
	}
	for _, ts := range tagSets {
		if ts.group == "" && ts.name == "" && typesMatch(ts.typ, typ) {
			return true
		}
	}
	return false
}

type provideNode struct {
	constructor any
	opts        []any
	// paramTagsOverride rewrites constructor params to target replacement values.
	paramTagsOverride []string
}

func (n provideNode) Build() (fx.Option, error) {
	cfg, _, extra, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, err
	}
	constructor := n.constructor
	if cfg.autoInjectFields && !cfg.ignoreAutoInjectFields {
		wrapped, ok, err := wrapAutoInjectConstructor(constructor)
		if err != nil {
			return nil, err
		}
		if ok {
			constructor = wrapped
		}
	}
	spec, _, err := buildProvideSpec(cfg, constructor, nil)
	if err != nil {
		return nil, err
	}
	finalConstructor := constructor
	if len(spec.exports) > 0 || spec.includeSelf {
		wrapped, err := buildGroupedConstructor(constructor, spec.exports, spec.includeSelf)
		if err != nil {
			return nil, err
		}
		finalConstructor = wrapped
	}
	if n.paramTagsOverride != nil {
		finalConstructor = fx.Annotate(finalConstructor, fx.ParamTags(n.paramTagsOverride...))
	}
	var provideOpt fx.Option
	if cfg.privateSet && cfg.privateValue {
		provideOpt = fx.Provide(finalConstructor, fx.Private)
	} else {
		provideOpt = fx.Provide(finalConstructor)
	}
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
	value := n.value
	var constructor any
	if constructor == nil && cfg.autoInjectFields && !cfg.ignoreAutoInjectFields {
		wrapped, ok, err := wrapAutoInjectSupply(value)
		if err != nil {
			return nil, err
		}
		if ok {
			constructor = wrapped
		}
	}
	spec, _, err := buildProvideSpec(cfg, constructor, value)
	if err != nil {
		return nil, err
	}
	var provideOpt fx.Option
	if constructor != nil {
		provideOpt, err = buildProvideConstructorOption(spec, constructor)
	} else {
		provideOpt, err = buildProvideSupplyOption(spec, value)
	}
	if err != nil {
		return nil, err
	}
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	if len(out) == 1 {
		return out[0], nil
	}
	return fx.Options(out...), nil
}
