package di

import (
	"fmt"
	"reflect"
	"sync"

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
		return nil, fmt.Errorf(errConstructorNil)
	}
	fn := reflect.TypeOf(constructor)
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf(errConstructorMustBeFunction)
	}
	numOut := fn.NumOut()
	if numOut < 1 || numOut > 2 {
		return nil, fmt.Errorf(errConstructorReturnCount)
	}
	if numOut == 2 && fn.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return nil, fmt.Errorf(errConstructorSecondResult)
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
				return nil, fmt.Errorf(errProvideValueNil)
			}
			return baseType, nil
		}
		return nil, fmt.Errorf(errCannotInferType)
	}
	if len(cfg.pendingNames) > 0 || len(cfg.pendingGroups) > 0 {
		if len(cfg.exports) > 0 {
			return provideSpec{}, nil, fmt.Errorf(errWithTagsSingleAs)
		}
		if _, err := getBaseType(); err != nil {
			return provideSpec{}, nil, err
		}
		for _, name := range cfg.pendingNames {
			spec.exports = append(spec.exports, exportSpec{
				typ:   baseType,
				name:  name,
				named: true,
			})
			tagSets = append(tagSets, tagSet{name: name, typ: baseType})
		}
		for _, group := range cfg.pendingGroups {
			spec.exports = append(spec.exports, exportSpec{
				typ:     baseType,
				group:   group,
				grouped: true,
			})
			tagSets = append(tagSets, tagSet{group: group, typ: baseType})
		}
	}

	hasUntagged := false
	noExplicitExports := len(cfg.exports) == 0 && len(cfg.pendingNames) == 0 && len(cfg.pendingGroups) == 0 && !cfg.includeSelf
	for _, exp := range cfg.exports {
		if exp.grouped && exp.named {
			return provideSpec{}, nil, fmt.Errorf(errExportCannotBeGroupedAndNamed)
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
			if isAutoGroupIgnored(cfg, rule) {
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

	if noExplicitExports && !spec.includeSelf && hasUntagged && len(cfg.autoGroups) > 0 && !cfg.ignoreAuto {
		// Auto-group-only providers do not need the untagged export.
		tagSets = tagSets[:0]
		hasUntagged = false
	}

	if spec.includeSelf && !hasUntagged && !tagSetHasType(tagSets, baseType) {
		if _, err := getBaseType(); err != nil {
			return provideSpec{}, nil, err
		}
		tagSets = append(tagSets, tagSet{typ: baseType})
	}

	return spec, tagSets, nil
}

func isAutoGroupIgnored(cfg bindConfig, rule autoGroupRule) bool {
	for _, ignore := range cfg.autoGroupIgnores {
		if ignore.iface != rule.iface {
			continue
		}
		if ignore.group != "" && ignore.group != rule.group {
			continue
		}
		return true
	}
	return false
}

type implementsKey struct {
	base  reflect.Type
	iface reflect.Type
}

var implementsCache sync.Map

func implementsInterface(base reflect.Type, iface reflect.Type) bool {
	if iface == nil || iface.Kind() != reflect.Interface {
		return false
	}
	if base == nil {
		return false
	}
	key := implementsKey{base: base, iface: iface}
	if cached, ok := implementsCache.Load(key); ok {
		return cached.(bool)
	}
	ok := base.Implements(iface)
	if !ok && base.Kind() != reflect.Pointer {
		ok = reflect.PointerTo(base).Implements(iface)
	}
	implementsCache.Store(key, ok)
	return ok
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
		// Wrap constructor to auto-inject fields before construction.
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
	if len(cfg.metadata) > 0 {
		wrapped, err := buildMetadataConstructor(constructor, spec.exports, spec.includeSelf, cfg.metadata)
		if err != nil {
			return nil, err
		}
		finalConstructor = wrapped
	} else if len(spec.exports) > 0 || spec.includeSelf {
		// Wrap constructor to emit multiple exports (As/Group/Name).
		wrapped, err := buildGroupedConstructor(constructor, spec.exports, spec.includeSelf)
		if err != nil {
			return nil, err
		}
		finalConstructor = wrapped
	}
	paramTags := n.paramTagsOverride
	if paramTags == nil {
		paramTags = cfg.paramTags
	}
	if hasAnyTag(paramTags) {
		// Apply positional param tags to the constructor.
		finalConstructor = fx.Annotate(finalConstructor, fx.ParamTags(paramTags...))
	}
	var provideOpt fx.Option
	if cfg.privateSet && cfg.privateValue {
		// Hide this provider from other modules.
		provideOpt = fx.Provide(finalConstructor, fx.Private)
	} else {
		provideOpt = fx.Provide(finalConstructor)
	}
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	return packOptions(out), nil
}

func (n provideNode) buildConstructor() (any, bool, []fx.Option, error) {
	cfg, _, extra, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, false, nil, err
	}
	constructor := n.constructor
	if cfg.autoInjectFields && !cfg.ignoreAutoInjectFields {
		wrapped, ok, err := wrapAutoInjectConstructor(constructor)
		if err != nil {
			return nil, false, nil, err
		}
		if ok {
			constructor = wrapped
		}
	}
	spec, _, err := buildProvideSpec(cfg, constructor, nil)
	if err != nil {
		return nil, false, nil, err
	}
	finalConstructor := constructor
	if len(cfg.metadata) > 0 {
		wrapped, err := buildMetadataConstructor(constructor, spec.exports, spec.includeSelf, cfg.metadata)
		if err != nil {
			return nil, false, nil, err
		}
		finalConstructor = wrapped
	} else if len(spec.exports) > 0 || spec.includeSelf {
		wrapped, err := buildGroupedConstructor(constructor, spec.exports, spec.includeSelf)
		if err != nil {
			return nil, false, nil, err
		}
		finalConstructor = wrapped
	}
	paramTags := n.paramTagsOverride
	if paramTags == nil {
		paramTags = cfg.paramTags
	}
	if hasAnyTag(paramTags) {
		finalConstructor = fx.Annotate(finalConstructor, fx.ParamTags(paramTags...))
	}
	private := cfg.privateSet && cfg.privateValue
	return finalConstructor, private, extra, nil
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
	if hasAnyTag(cfg.paramTags) {
		return nil, fmt.Errorf(errParamsNotSupportedWithSupply)
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
		provideOpt, err = buildProvideConstructorOption(spec, constructor, cfg.metadata)
	} else {
		provideOpt, err = buildProvideSupplyOption(spec, value, cfg.metadata)
	}
	if err != nil {
		return nil, err
	}
	var out []fx.Option
	out = append(out, provideOpt)
	out = append(out, extra...)
	return packOptions(out), nil
}

func (n supplyNode) buildSupply() (any, any, bool, bool, []fx.Option, error) {
	cfg, _, extra, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, nil, false, false, nil, err
	}
	if hasAnyTag(cfg.paramTags) {
		return nil, nil, false, false, nil, fmt.Errorf(errParamsNotSupportedWithSupply)
	}
	value := n.value
	var constructor any
	if constructor == nil && cfg.autoInjectFields && !cfg.ignoreAutoInjectFields {
		wrapped, ok, err := wrapAutoInjectSupply(value)
		if err != nil {
			return nil, nil, false, false, nil, err
		}
		if ok {
			constructor = wrapped
		}
	}
	spec, _, err := buildProvideSpec(cfg, constructor, value)
	if err != nil {
		return nil, nil, false, false, nil, err
	}
	private := cfg.privateSet && cfg.privateValue
	if constructor != nil {
		finalConstructor := constructor
		if len(cfg.metadata) > 0 {
			wrapped, err := buildMetadataConstructor(constructor, spec.exports, spec.includeSelf, cfg.metadata)
			if err != nil {
				return nil, nil, false, false, nil, err
			}
			finalConstructor = wrapped
		} else if len(spec.exports) > 0 || spec.includeSelf {
			wrapped, err := buildGroupedConstructor(constructor, spec.exports, spec.includeSelf)
			if err != nil {
				return nil, nil, false, false, nil, err
			}
			finalConstructor = wrapped
		}
		return finalConstructor, nil, false, private, extra, nil
	}
	if len(cfg.metadata) > 0 {
		wrapped, err := buildMetadataSupply(value, spec.exports, spec.includeSelf, cfg.metadata)
		if err != nil {
			return nil, nil, false, false, nil, err
		}
		return wrapped, nil, false, private, extra, nil
	}
	if len(spec.exports) > 0 || spec.includeSelf {
		wrapped, err := buildGroupedSupply(value, spec.exports, spec.includeSelf)
		if err != nil {
			return nil, nil, false, false, nil, err
		}
		return wrapped, nil, false, private, extra, nil
	}
	return nil, value, true, private, extra, nil
}

func hasAnyTag(tags []string) bool {
	for _, tag := range tags {
		if tag != "" {
			return true
		}
	}
	return false
}
