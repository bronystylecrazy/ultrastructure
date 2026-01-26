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

	for _, entry := range entries {
		explicit, hasExplicit, err := explicitTagSets(entry.dec)
		if err != nil {
			return nil, err
		}
		targets := entry.tagSets
		if hasExplicit {
			targets = explicit
		}

		fnType := reflect.TypeOf(entry.dec.function)
		if fnType == nil || fnType.Kind() != reflect.Func {
			return nil, fmt.Errorf("decorate must be a function")
		}
		isSliceParam := fnType.NumIn() > 0 && fnType.In(0).Kind() == reflect.Slice

		for _, ts := range targets {
			if ts.group != "" && !isSliceParam {
				if ts.name == "" {
					continue
				}
				// fall back to name-only for non-slice decorators
				ts = tagSet{name: ts.name}
			}
			key := tagSetKey(ts)
			b := buckets[key]
			if b == nil {
				b = &bucket{ts: ts}
				buckets[key] = b
			}
			b.funcs = append(b.funcs, entry.dec.function)
		}
	}

	var out []fx.Option
	for _, b := range buckets {
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
	for _, opt := range dec.opts {
		if opt != nil {
			opt.applyParam(&cfg)
		}
		if cfg.err != nil {
			return nil, false, cfg.err
		}
	}
	if len(cfg.tags) == 0 {
		return nil, false, nil
	}
	if len(cfg.tags) > 1 {
		return nil, false, fmt.Errorf("decorate supports only one name/group tag")
	}
	t := cfg.tags[0]
	if len(t) >= 6 && t[:5] == "name:" {
		return []tagSet{{name: t[6 : len(t)-1]}}, true, nil
	}
	if len(t) >= 7 && t[:6] == "group:" {
		return []tagSet{{group: t[7 : len(t)-1]}}, true, nil
	}
	return nil, false, fmt.Errorf("unsupported decorate tag %q", t)
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
		return nil, fmt.Errorf("decorate must be a function")
	}
	if base.NumIn() != 1 {
		return nil, fmt.Errorf("decorate functions must take exactly one argument")
	}
	if base.NumOut() != 1 && base.NumOut() != 2 {
		return nil, fmt.Errorf("decorate functions must return 1 value (and optional error)")
	}
	if base.NumOut() == 2 && base.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return nil, fmt.Errorf("decorate function's second result must be error")
	}

	for i := 1; i < len(funcs); i++ {
		fn := reflect.TypeOf(funcs[i])
		if fn == nil || fn.Kind() != reflect.Func {
			return nil, fmt.Errorf("decorate must be a function")
		}
		if fn.NumIn() != base.NumIn() || fn.NumOut() != base.NumOut() {
			return nil, fmt.Errorf("decorate functions must match signature")
		}
		if fn.In(0) != base.In(0) || fn.Out(0) != base.Out(0) {
			return nil, fmt.Errorf("decorate functions must match signature")
		}
		if fn.NumOut() == 2 && fn.Out(1) != base.Out(1) {
			return nil, fmt.Errorf("decorate functions must match signature")
		}
	}

	fn := reflect.MakeFunc(base, func(args []reflect.Value) []reflect.Value {
		in := args[0]
		for _, f := range funcs {
			out := reflect.ValueOf(f).Call([]reflect.Value{in})
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

type decorateNode struct {
	function any
	opts     []Option
}

func (n decorateNode) Build() (fx.Option, error) {
	var cfg paramConfig
	for _, opt := range n.opts {
		if opt != nil {
			opt.applyParam(&cfg)
		}
		if cfg.err != nil {
			return nil, cfg.err
		}
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
