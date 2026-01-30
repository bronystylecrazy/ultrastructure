package di

import (
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

// Decorate declares a decorate node.
func Decorate(function any, opts ...Option) Node {
	return decorateNode{function: function, opts: opts}
}

func buildDecorators(entries []decorateEntry) ([]fx.Option, error) {
	type bucket struct {
		ts    tagSet
		funcs []any
	}
	buckets := map[string]*bucket{}
	order := []string{}

	for _, entry := range entries {
		explicit, hasExplicit, err := explicitTagSets(entry.dec)
		if err != nil {
			return nil, err
		}
		// Use explicit tags if provided; otherwise apply to all matched tag sets.
		targets := entry.tagSets
		if hasExplicit {
			targets = explicit
		}

		fnType := reflect.TypeOf(entry.dec.function)
		if fnType == nil || fnType.Kind() != reflect.Func {
			return nil, fmt.Errorf(errDecorateFunctionRequired)
		}
		// Slice decorators can target groups directly.
		isSliceParam := fnType.NumIn() > 0 && fnType.In(0).Kind() == reflect.Slice

		for _, ts := range targets {
			if ts.group != "" && !isSliceParam {
				if ts.name == "" {
					continue
				}
				// fall back to name-only for non-slice decorators
				ts = tagSet{name: ts.name}
			}
			key := tagSetKey(ts) + "|sig:" + fnType.String()
			b := buckets[key]
			if b == nil {
				b = &bucket{ts: ts}
				buckets[key] = b
				order = append(order, key)
			}
			b.funcs = append(b.funcs, entry.dec.function)
		}
	}

	var out []fx.Option
	for _, key := range order {
		b := buckets[key]
		if len(b.funcs) == 0 {
			continue
		}
		fn, err := composeDecorators(b.funcs)
		if err != nil {
			return nil, err
		}
		if b.ts.name == "" && b.ts.group == "" {
			out = append(out, fx.Decorate(fn))
			continue
		}
		tags := tagSetTags(b.ts)
		anns := []fx.Annotation{fx.ParamTags(tags...), fx.ResultTags(tags...)}
		out = append(out, fx.Decorate(fx.Annotate(fn, anns...)))
	}
	return out, nil
}

func explicitTagSets(dec decorateNode) ([]tagSet, bool, error) {
	var cfg paramConfig
	if err := applyParamOptions(dec.opts, &cfg); err != nil {
		return nil, false, err
	}
	if len(cfg.tags) == 0 {
		return nil, false, nil
	}
	if len(cfg.tags) > 1 {
		return nil, false, fmt.Errorf(errDecorateNameGroupSingle)
	}
	t := cfg.tags[0]
	if len(t) >= 6 && t[:5] == "name:" {
		return []tagSet{{name: t[6 : len(t)-1]}}, true, nil
	}
	if len(t) >= 7 && t[:6] == "group:" {
		return []tagSet{{group: t[7 : len(t)-1]}}, true, nil
	}
	return nil, false, fmt.Errorf(errUnsupportedTag, t)
}

func tagSetTags(ts tagSet) []string {
	if ts.name != "" {
		return []string{`name:"` + ts.name + `"`}
	}
	if ts.group != "" {
		return []string{`group:"` + ts.group + `"`}
	}
	return nil
}

func tagSetKey(ts tagSet) string {
	if ts.name != "" {
		return "n:" + ts.name
	}
	if ts.group != "" {
		return "g:" + ts.group
	}
	return ""
}

func composeDecorators(funcs []any) (any, error) {
	if len(funcs) == 1 {
		return funcs[0], nil
	}
	base := reflect.TypeOf(funcs[0])
	if base == nil || base.Kind() != reflect.Func {
		return nil, fmt.Errorf(errDecorateFunctionRequired)
	}
	if err := validateDecoratorSignature(base, base); err != nil {
		return nil, err
	}

	for i := 1; i < len(funcs); i++ {
		fn := reflect.TypeOf(funcs[i])
		if fn == nil || fn.Kind() != reflect.Func {
			return nil, fmt.Errorf(errDecorateFunctionRequired)
		}
		if err := validateDecoratorSignature(base, fn); err != nil {
			return nil, fmt.Errorf(errDecorateSignatureMismatch)
		}
	}

	// Build a composite decorator that chains each function in order.
	fn := reflect.MakeFunc(base, func(args []reflect.Value) []reflect.Value {
		in := args[0]
		for _, f := range funcs {
			callArgs := make([]reflect.Value, len(args))
			callArgs[0] = in
			if len(args) > 1 {
				copy(callArgs[1:], args[1:])
			}
			out := reflect.ValueOf(f).Call(callArgs)
			in = out[0]
			if len(out) == 2 {
				if errVal := out[1]; !errVal.IsNil() {
					return []reflect.Value{in, errVal}
				}
			}
		}
		if base.NumOut() == 2 {
			return []reflect.Value{in, reflect.Zero(base.Out(1))}
		}
		return []reflect.Value{in}
	})
	return fn.Interface(), nil
}

func validateDecoratorSignature(base reflect.Type, fn reflect.Type) error {
	if fn.NumIn() < 1 {
		return fmt.Errorf(errDecorateTooFewArgs)
	}
	if fn.NumOut() != 1 && fn.NumOut() != 2 {
		return fmt.Errorf(errDecorateReturnCount)
	}
	if fn.NumOut() == 2 && fn.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf(errDecorateSecondResult)
	}
	if base != fn {
		if fn.NumIn() != base.NumIn() || fn.NumOut() != base.NumOut() {
			return fmt.Errorf(errDecorateSignatureMismatch)
		}
		if fn.In(0) != base.In(0) || fn.Out(0) != base.Out(0) {
			return fmt.Errorf(errDecorateSignatureMismatch)
		}
		for j := 1; j < fn.NumIn(); j++ {
			if fn.In(j) != base.In(j) {
				return fmt.Errorf(errDecorateSignatureMismatch)
			}
		}
		if fn.NumOut() == 2 && fn.Out(1) != base.Out(1) {
			return fmt.Errorf(errDecorateSignatureMismatch)
		}
	}
	return nil
}

type decorateNode struct {
	function any
	opts     []Option
}

func (n decorateNode) Build() (fx.Option, error) {
	var cfg paramConfig
	if err := applyParamOptions(n.opts, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.tags) == 0 && len(cfg.resultTags) == 0 {
		return fx.Decorate(n.function), nil
	}
	anns := []fx.Annotation{}
	if len(cfg.tags) > 0 {
		anns = append(anns, fx.ParamTags(cfg.tags...))
	}
	if len(cfg.resultTags) > 0 {
		anns = append(anns, fx.ResultTags(cfg.resultTags...))
	}
	return fx.Decorate(fx.Annotate(n.function, anns...)), nil
}
