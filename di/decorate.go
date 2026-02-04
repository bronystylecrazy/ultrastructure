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

type decorateSpec struct {
	fn        any
	fnType    reflect.Type
	extraTags []string
	depIdx    []int
}

type decorateBucket struct {
	ts        tagSet
	targetTyp reflect.Type
	specs     []decorateSpec
}

func buildDecorators(entries []decorateEntry) ([]fx.Option, error) {
	buckets := map[string]*decorateBucket{}
	order := []string{}

	for _, entry := range entries {
		fnType := reflect.TypeOf(entry.dec.function)
		if fnType == nil || fnType.Kind() != reflect.Func {
			return nil, fmt.Errorf(errDecorateFunctionRequired)
		}
		if fnType.NumIn() < 1 {
			return nil, fmt.Errorf(errDecorateTooFewArgs)
		}
		if fnType.NumIn() == 1 && isFxInStruct(fnType.In(0)) {
			return nil, fmt.Errorf(errDecorateSignatureMismatch)
		}
		if hasFxInParam(fnType) {
			var cfg paramConfig
			if err := applyParamOptions(entry.dec.opts, &cfg); err != nil {
				return nil, err
			}
			if len(cfg.tags) > 0 && !tagsEqual(cfg.tags, cfg.resultTags) {
				return nil, fmt.Errorf(errDecorateSignatureMismatch)
			}
		}
		explicit, hasExplicit, err := explicitTagSets(entry.dec, fnType)
		if err != nil {
			return nil, err
		}
		// Use explicit tags if provided; otherwise apply to all matched tag sets.
		targets := entry.tagSets
		if hasExplicit {
			targets = explicit
		}
		targetType, isSliceTarget := decoratorTargetType(fnType)
		// Slice decorators can target groups directly.
		isSliceParam := isSliceTarget

		for _, ts := range targets {
			if ts.typ == nil {
				ts.typ = targetType
			}
			if !decoratorTargetsTagSet(ts, targetType, isSliceParam) {
				continue
			}
			fn := entry.dec.function
			fnSig := fnType
			bucketTarget := targetType
			if ts.group != "" && !isSliceParam {
				if ts.name != "" {
					// fall back to name-only for non-slice decorators
					ts = tagSet{name: ts.name}
				} else {
					wrapped, err := buildGroupDecoratorWrapper(entry.dec.function, ts.typ)
					if err != nil {
						return nil, err
					}
					if wrapped == nil {
						continue
					}
					fn = wrapped
					fnSig = reflect.TypeOf(fn)
					if wrappedTarget, _ := decoratorTargetType(fnSig); wrappedTarget != nil {
						bucketTarget = wrappedTarget
					}
				}
			}
			extraTags, err := buildDecorateExtraTags(entry.dec, fnSig)
			if err != nil {
				return nil, err
			}
			key := tagSetKey(ts) + "|t:" + bucketTarget.String()
			b := buckets[key]
			if b == nil {
				b = &decorateBucket{ts: ts, targetTyp: fnSig.In(0)}
				buckets[key] = b
				order = append(order, key)
			}
			b.specs = append(b.specs, decorateSpec{
				fn:        fn,
				fnType:    fnSig,
				extraTags: extraTags,
			})
		}
	}

	var out []fx.Option
	for _, key := range order {
		b := buckets[key]
		if b == nil || len(b.specs) == 0 {
			continue
		}
		if len(b.specs) == 1 {
			spec := b.specs[0]
			paramTags := buildDecorateParamTagsFromSpec(spec, b.ts)
			resultTags := buildDecorateResultTags(b.ts)
			if len(paramTags) == 0 && len(resultTags) == 0 {
				out = append(out, fx.Decorate(spec.fn))
				continue
			}
			anns := []fx.Annotation{}
			if len(paramTags) > 0 {
				anns = append(anns, fx.ParamTags(paramTags...))
			}
			if len(resultTags) > 0 {
				anns = append(anns, fx.ResultTags(resultTags...))
			}
			out = append(out, fx.Decorate(fx.Annotate(spec.fn, anns...)))
			continue
		}
		opt, err := buildCompositeDecorator(b)
		if err != nil {
			return nil, err
		}
		out = append(out, opt)
	}
	return out, nil
}

func decoratorTargetsTagSet(ts tagSet, target reflect.Type, isSlice bool) bool {
	if target == nil || ts.typ == nil {
		return false
	}
	if isSlice {
		if ts.group == "" {
			return false
		}
	}
	if ts.group != "" {
		return target.AssignableTo(ts.typ) || ts.typ.AssignableTo(target)
	}
	return ts.typ.AssignableTo(target)
}

func decoratorTargetType(fnType reflect.Type) (reflect.Type, bool) {
	if fnType == nil || fnType.Kind() != reflect.Func {
		return nil, false
	}
	var target reflect.Type
	if fnType.NumOut() > 0 {
		target = fnType.Out(0)
	} else if fnType.NumIn() > 0 {
		target = fnType.In(0)
	}
	if target == nil {
		return nil, false
	}
	if target.Kind() == reflect.Slice {
		return target.Elem(), true
	}
	return target, false
}

func explicitTagSets(dec decorateNode, fnType reflect.Type) ([]tagSet, bool, error) {
	var cfg paramConfig
	if err := applyParamOptions(dec.opts, &cfg); err != nil {
		return nil, false, err
	}
	if len(cfg.tags) > 0 && fnType != nil && fnType.NumIn() == 1 {
		if len(cfg.tags) > 1 {
			return nil, false, fmt.Errorf(errDecorateNameGroupSingle)
		}
		name, group := parseTagNameGroup(cfg.tags[0])
		if name != "" {
			return []tagSet{{name: name}}, true, nil
		}
		if group != "" {
			return []tagSet{{group: group}}, true, nil
		}
	}
	if !shouldUseResultTags(cfg, fnType) {
		return nil, false, nil
	}
	if len(cfg.resultTags) > 1 {
		return nil, false, fmt.Errorf(errDecorateNameGroupSingle)
	}
	t := cfg.resultTags[0]
	if len(t) >= 6 && t[:5] == "name:" {
		return []tagSet{{name: t[6 : len(t)-1]}}, true, nil
	}
	if len(t) >= 7 && t[:6] == "group:" {
		return []tagSet{{group: t[7 : len(t)-1]}}, true, nil
	}
	return nil, false, fmt.Errorf(errUnsupportedTag, t)
}

func buildDecorateExtraTags(dec decorateNode, fnType reflect.Type) ([]string, error) {
	var cfg paramConfig
	if err := applyParamOptions(dec.opts, &cfg); err != nil {
		return nil, err
	}
	extraTags := cfg.tags
	if shouldUseResultTags(cfg, fnType) && len(cfg.resultTags) > 0 && len(cfg.tags) > 0 && cfg.tags[0] == cfg.resultTags[0] {
		extraTags = cfg.tags[1:]
	}
	if fnType.NumIn() <= 1 {
		return nil, nil
	}
	tags := make([]string, fnType.NumIn()-1)
	for i := 0; i < len(tags) && i < len(extraTags); i++ {
		tags[i] = extraTags[i]
	}
	return tags, nil
}

func shouldUseResultTags(cfg paramConfig, fnType reflect.Type) bool {
	if len(cfg.resultTags) == 0 {
		return false
	}
	if fnType != nil && fnType.NumIn() > 1 && tagsEqual(cfg.tags, cfg.resultTags) {
		return false
	}
	return true
}

func tagsEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func buildDecorateParamTagsFromSpec(spec decorateSpec, ts tagSet) []string {
	fnType := spec.fnType
	if fnType == nil || fnType.NumIn() == 0 {
		return nil
	}
	if hasFxInParam(fnType) {
		return nil
	}
	paramTags := make([]string, fnType.NumIn())
	if ts.name != "" || ts.group != "" {
		paramTags[0] = rewriteParamTag("", ts)
	}
	extraIdx := 0
	for i := 1; i < fnType.NumIn() && extraIdx < len(spec.extraTags); i++ {
		paramTags[i] = spec.extraTags[extraIdx]
		extraIdx++
	}
	if !hasAnyTag(paramTags) {
		return nil
	}
	return paramTags
}

func buildDecorateResultTags(ts tagSet) []string {
	if ts.name != "" || ts.group != "" {
		return tagSetTags(ts)
	}
	return nil
}

func hasFxInParam(fnType reflect.Type) bool {
	for i := 0; i < fnType.NumIn(); i++ {
		param := fnType.In(i)
		if isFxInStruct(param) {
			return true
		}
	}
	return false
}

func isFxInStruct(t reflect.Type) bool {
	if t == nil || t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type == reflect.TypeOf(fx.In{}) {
			return true
		}
	}
	return false
}

func buildCompositeDecorator(b *decorateBucket) (fx.Option, error) {
	type depField struct {
		typ reflect.Type
		tag string
		idx int
	}
	depFields := []depField{}
	depIndex := map[string]int{}
	hasError := false
	for i := range b.specs {
		spec := &b.specs[i]
		if spec.fnType.NumOut() == 2 {
			hasError = true
		}
		if hasFxInParam(spec.fnType) {
			return nil, fmt.Errorf(errDecorateSignatureMismatch)
		}
		spec.depIdx = make([]int, 0, spec.fnType.NumIn()-1)
		for j := 1; j < spec.fnType.NumIn(); j++ {
			tag := ""
			if j-1 < len(spec.extraTags) {
				tag = spec.extraTags[j-1]
			}
			key := spec.fnType.In(j).String() + "|" + tag
			idx, ok := depIndex[key]
			if !ok {
				idx = len(depFields)
				depIndex[key] = idx
				depFields = append(depFields, depField{typ: spec.fnType.In(j), tag: tag, idx: idx})
			}
			spec.depIdx = append(spec.depIdx, idx)
		}
	}

	fields := []reflect.StructField{{
		Name:      "In",
		Type:      reflect.TypeOf(fx.In{}),
		Anonymous: true,
	}}
	targetTag := ""
	if b.ts.name != "" {
		targetTag = `name:"` + b.ts.name + `"`
	} else if b.ts.group != "" {
		targetTag = `group:"` + b.ts.group + `"`
	}
	fields = append(fields, reflect.StructField{
		Name: "Target",
		Type: b.targetTyp,
		Tag:  reflect.StructTag(targetTag),
	})
	for i, dep := range depFields {
		fields = append(fields, reflect.StructField{
			Name: fmt.Sprintf("Dep%d", i),
			Type: dep.typ,
			Tag:  reflect.StructTag(dep.tag),
		})
	}

	inType := reflect.StructOf(fields)
	outTypes := []reflect.Type{b.targetTyp}
	if hasError {
		outTypes = append(outTypes, errorType)
	}
	wrapperType := reflect.FuncOf([]reflect.Type{inType}, outTypes, false)
	wrapper := reflect.MakeFunc(wrapperType, func(args []reflect.Value) []reflect.Value {
		in := args[0]
		current := in.Field(1)
		for _, spec := range b.specs {
			callArgs := make([]reflect.Value, spec.fnType.NumIn())
			callArgs[0] = current
			for j := 1; j < spec.fnType.NumIn(); j++ {
				fieldIdx := 2 + spec.depIdx[j-1]
				callArgs[j] = in.Field(fieldIdx)
			}
			results := reflect.ValueOf(spec.fn).Call(callArgs)
			current = results[0]
			if spec.fnType.NumOut() == 2 && !results[1].IsNil() {
				return []reflect.Value{current, results[1]}
			}
		}
		if hasError {
			return []reflect.Value{current, reflect.Zero(errorType)}
		}
		return []reflect.Value{current}
	})
	return fx.Decorate(wrapper.Interface()), nil
}

func buildGroupDecoratorWrapper(fn any, groupIface reflect.Type) (any, error) {
	if groupIface == nil || groupIface.Kind() != reflect.Interface {
		return nil, nil
	}
	fnType := reflect.TypeOf(fn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf(errDecorateFunctionRequired)
	}
	if fnType.NumIn() < 1 {
		return nil, fmt.Errorf(errDecorateTooFewArgs)
	}
	if fnType.NumOut() != 1 && fnType.NumOut() != 2 {
		return nil, fmt.Errorf(errDecorateReturnCount)
	}
	if fnType.NumOut() == 2 && fnType.Out(1) != errorType {
		return nil, fmt.Errorf(errDecorateSecondResult)
	}
	inTypes := []reflect.Type{reflect.SliceOf(groupIface)}
	for i := 1; i < fnType.NumIn(); i++ {
		inTypes = append(inTypes, fnType.In(i))
	}
	outTypes := []reflect.Type{reflect.SliceOf(groupIface)}
	if fnType.NumOut() == 2 {
		outTypes = append(outTypes, errorType)
	}
	wrapperType := reflect.FuncOf(inTypes, outTypes, fnType.IsVariadic())
	orig := reflect.ValueOf(fn)
	wrapper := reflect.MakeFunc(wrapperType, func(args []reflect.Value) []reflect.Value {
		inSlice := args[0]
		outSlice := reflect.MakeSlice(inSlice.Type(), inSlice.Len(), inSlice.Len())
		for i := 0; i < inSlice.Len(); i++ {
			elem := inSlice.Index(i)
			outElem := elem
			if v, ok := coerceDecoratorValue(elem, fnType.In(0)); ok {
				callArgs := make([]reflect.Value, fnType.NumIn())
				callArgs[0] = v
				if len(args) > 1 {
					copy(callArgs[1:], args[1:])
				}
				results := orig.Call(callArgs)
				if fnType.NumOut() == 2 && !results[1].IsNil() {
					return []reflect.Value{inSlice, results[1]}
				}
				outElem = results[0]
			}
			if !outElem.Type().AssignableTo(groupIface) {
				if outElem.Type().ConvertibleTo(groupIface) {
					outElem = outElem.Convert(groupIface)
				} else {
					outElem = elem
				}
			}
			outSlice.Index(i).Set(outElem)
		}
		if fnType.NumOut() == 2 {
			return []reflect.Value{outSlice, reflect.Zero(errorType)}
		}
		return []reflect.Value{outSlice}
	})
	return wrapper.Interface(), nil
}

func coerceDecoratorValue(elem reflect.Value, target reflect.Type) (reflect.Value, bool) {
	if !elem.IsValid() {
		return reflect.Value{}, false
	}
	if elem.Type().AssignableTo(target) {
		if elem.Type() == target {
			return elem, true
		}
		if elem.Type().ConvertibleTo(target) {
			return elem.Convert(target), true
		}
		return elem, true
	}
	if elem.Kind() == reflect.Interface && !elem.IsNil() {
		concrete := elem.Elem()
		if concrete.Type().AssignableTo(target) {
			if concrete.Type() == target {
				return concrete, true
			}
			if concrete.Type().ConvertibleTo(target) {
				return concrete.Convert(target), true
			}
			return concrete, true
		}
	}
	return reflect.Value{}, false
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
