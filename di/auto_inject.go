package di

import (
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/fx"
)

// AutoInject enables automatic injection into tagged struct fields.
// Supported tags:
//   di:"inject"
//   di:"name=prod"
//   di:"group=loggers"
//   di:"optional"
// Tags can be comma-separated.
func AutoInject() Node {
	return autoInjectNode{}
}

type autoInjectNode struct{}

func (n autoInjectNode) Build() (fx.Option, error) {
	return fx.Options(), nil
}

type autoInjectOption struct{}

func (o autoInjectOption) applyBind(cfg *bindConfig)  { cfg.autoInjectFields = true }
func (o autoInjectOption) applyParam(*paramConfig)    {}

type autoInjectApplier interface {
	withAutoInjectFields(bool) Node
}

func applyAutoInjectFields(nodes []Node, enabled bool) []Node {
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		switch v := n.(type) {
		case autoInjectNode:
			enabled = true
			out[i] = v
		case moduleNode:
			v.nodes = applyAutoInjectFields(v.nodes, enabled)
			out[i] = v
		case optionsNode:
			v.nodes = applyAutoInjectFields(v.nodes, enabled)
			out[i] = v
		case conditionalNode:
			v.nodes = applyAutoInjectFields(v.nodes, enabled)
			out[i] = v
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for idx, c := range v.cases {
				c.nodes = applyAutoInjectFields(c.nodes, enabled)
				cases[idx] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: applyAutoInjectFields(v.defaultCase.nodes, enabled)}
			out[i] = v
		default:
			if applier, ok := n.(autoInjectApplier); ok {
				out[i] = applier.withAutoInjectFields(enabled)
			} else {
				out[i] = n
			}
		}
	}
	return out
}

func (n provideNode) withAutoInjectFields(enabled bool) Node {
	if !enabled {
		return n
	}
	opts := append([]any{}, n.opts...)
	opts = append(opts, autoInjectOption{})
	return provideNode{constructor: n.constructor, opts: opts}
}

func (n supplyNode) withAutoInjectFields(enabled bool) Node {
	if !enabled {
		return n
	}
	opts := append([]any{}, n.opts...)
	opts = append(opts, autoInjectOption{})
	return supplyNode{value: n.value, opts: opts}
}

type injectFieldSpec struct {
	fieldIndex int
	fieldType  reflect.Type
	tag        reflect.StructTag
}

func parseInjectTag(tag string) (reflect.StructTag, bool) {
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	var name, group, optional string
	inject := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == "inject" {
			inject = true
			continue
		}
		if part == "optional" {
			optional = `optional:"true"`
			inject = true
			continue
		}
		if strings.HasPrefix(part, "name=") {
			name = strings.TrimPrefix(part, "name=")
			inject = true
			continue
		}
		if strings.HasPrefix(part, "group=") {
			group = strings.TrimPrefix(part, "group=")
			inject = true
			continue
		}
	}
	if !inject {
		return "", false
	}
	var tags []string
	if name != "" {
		tags = append(tags, `name:"`+name+`"`)
	}
	if group != "" {
		tags = append(tags, `group:"`+group+`"`)
	}
	if optional != "" {
		tags = append(tags, optional)
	}
	return reflect.StructTag(strings.Join(tags, " ")), true
}

func findInjectFields(t reflect.Type) []injectFieldSpec {
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var fields []injectFieldSpec
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag, ok := parseInjectTag(field.Tag.Get("di"))
		if !ok {
			continue
		}
		fields = append(fields, injectFieldSpec{
			fieldIndex: i,
			fieldType:  field.Type,
			tag:        tag,
		})
	}
	return fields
}

func applyInjectFields(value reflect.Value, specs []injectFieldSpec, args []reflect.Value) reflect.Value {
	if len(specs) == 0 {
		return value
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return value
		}
		elem := value.Elem()
		if elem.Kind() != reflect.Struct {
			return value
		}
		for i, spec := range specs {
			field := elem.Field(spec.fieldIndex)
			if field.IsValid() && field.CanSet() {
				field.Set(args[i])
			}
		}
		return value
	}
	if value.Kind() == reflect.Struct {
		copyVal := reflect.New(value.Type()).Elem()
		copyVal.Set(value)
		for i, spec := range specs {
			field := copyVal.Field(spec.fieldIndex)
			if field.IsValid() && field.CanSet() {
				field.Set(args[i])
			}
		}
		return copyVal
	}
	return value
}

func wrapAutoInjectConstructor(fn any) (any, bool, error) {
	if fn == nil {
		return fn, false, nil
	}
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func || fnType.IsVariadic() {
		return fn, false, nil
	}
	resType, err := constructorResultType(fn)
	if err != nil {
		return nil, false, err
	}
	specs := findInjectFields(resType)
	if len(specs) == 0 {
		return fn, false, nil
	}

	fields := []reflect.StructField{{
		Name:      "In",
		Type:      reflect.TypeOf(fx.In{}),
		Anonymous: true,
	}}
	for i := 0; i < fnType.NumIn(); i++ {
		fields = append(fields, reflect.StructField{
			Name: fmt.Sprintf("Arg%d", i),
			Type: fnType.In(i),
		})
	}
	for i, spec := range specs {
		fields = append(fields, reflect.StructField{
			Name: fmt.Sprintf("Inject%d", i),
			Type: spec.fieldType,
			Tag:  spec.tag,
		})
	}

	inType := reflect.StructOf(fields)
	outTypes := []reflect.Type{fnType.Out(0)}
	if fnType.NumOut() == 2 {
		outTypes = append(outTypes, fnType.Out(1))
	}

	orig := reflect.ValueOf(fn)
	wrapperType := reflect.FuncOf([]reflect.Type{inType}, outTypes, false)
	wrapper := reflect.MakeFunc(wrapperType, func(args []reflect.Value) []reflect.Value {
		in := args[0]
		callArgs := make([]reflect.Value, 0, fnType.NumIn())
		for i := 0; i < fnType.NumIn(); i++ {
			callArgs = append(callArgs, in.Field(i+1))
		}
		results := orig.Call(callArgs)
		if fnType.NumOut() == 2 {
			if errVal := results[1]; !errVal.IsNil() {
				return results
			}
		}
		injectArgs := make([]reflect.Value, 0, len(specs))
		for i := 0; i < len(specs); i++ {
			injectArgs = append(injectArgs, in.Field(fnType.NumIn()+1+i))
		}
		results[0] = applyInjectFields(results[0], specs, injectArgs)
		return results
	})
	return wrapper.Interface(), true, nil
}

func wrapAutoInjectSupply(value any) (any, bool, error) {
	if value == nil {
		return nil, false, nil
	}
	val := reflect.ValueOf(value)
	typ := val.Type()
	specs := findInjectFields(typ)
	if len(specs) == 0 {
		return value, false, nil
	}

	inTypes := make([]reflect.Type, len(specs))
	for i, spec := range specs {
		inTypes[i] = spec.fieldType
	}
	fnType := reflect.FuncOf(inTypes, []reflect.Type{typ}, false)
	fn := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		out := val
		out = applyInjectFields(out, specs, args)
		return []reflect.Value{out}
	})
	return fn.Interface(), true, nil
}
