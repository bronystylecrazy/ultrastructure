package di

import (
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func buildProvideConstructorOption(spec provideSpec, constructor any) (fx.Option, error) {
	if len(spec.exports) == 0 && !spec.includeSelf {
		if spec.privateSet && spec.privateValue {
			return fx.Provide(constructor, fx.Private), nil
		}
		return fx.Provide(constructor), nil
	}
	wrapped, err := buildGroupedConstructor(constructor, spec.exports, spec.includeSelf)
	if err != nil {
		return nil, err
	}
	if spec.privateSet && spec.privateValue {
		return fx.Provide(wrapped, fx.Private), nil
	}
	return fx.Provide(wrapped), nil
}

func buildProvideSupplyOption(spec provideSpec, value any) (fx.Option, error) {
	if len(spec.exports) == 0 && !spec.includeSelf {
		if spec.privateSet && spec.privateValue {
			return fx.Supply(value, fx.Private), nil
		}
		return fx.Supply(value), nil
	}
	wrapped, err := buildGroupedSupply(value, spec.exports, spec.includeSelf)
	if err != nil {
		return nil, err
	}
	if spec.privateSet && spec.privateValue {
		return fx.Provide(wrapped, fx.Private), nil
	}
	return fx.Provide(wrapped), nil
}

func buildGroupedConstructor(constructor any, exports []exportSpec, includeSelf bool) (any, error) {
	if constructor == nil {
		return nil, fmt.Errorf("constructor must not be nil")
	}
	if len(exports) == 0 && !includeSelf {
		return constructor, nil
	}
	fn := reflect.TypeOf(constructor)
	if fn.Kind() != reflect.Func {
		return nil, fmt.Errorf("constructor must be a function, got %v", fn)
	}
	numOut := fn.NumOut()
	if numOut < 1 || numOut > 2 {
		return nil, fmt.Errorf("constructor must return 1 value (and optional error), got %d results", numOut)
	}
	hasErr := false
	if numOut == 2 {
		if fn.Out(1) != errorType {
			return nil, fmt.Errorf("constructor's second result must be error")
		}
		hasErr = true
	}
	valueType := fn.Out(0)
	if err := validateExports(valueType, exports); err != nil {
		return nil, err
	}
	fields := []reflect.StructField{{
		Name:      "Out",
		Type:      reflect.TypeOf(fx.Out{}),
		Anonymous: true,
	}}
	if includeSelf {
		fields = append(fields, reflect.StructField{
			Name: "Self",
			Type: valueType,
		})
	}
	for i, exp := range exports {
		tag := reflect.StructTag("")
		if exp.grouped {
			tag = reflect.StructTag(`group:"` + exp.group + `"`)
		} else if exp.named {
			tag = reflect.StructTag(`name:"` + exp.name + `"`)
		}
		fields = append(fields, reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: exp.typ,
			Tag:  tag,
		})
	}
	outType := reflect.StructOf(fields)

	inTypes := make([]reflect.Type, fn.NumIn())
	for i := 0; i < fn.NumIn(); i++ {
		inTypes[i] = fn.In(i)
	}
	outTypes := []reflect.Type{outType}
	if hasErr {
		outTypes = append(outTypes, errorType)
	}
	orig := reflect.ValueOf(constructor)
	wrapperType := reflect.FuncOf(inTypes, outTypes, fn.IsVariadic())
	wrapper := reflect.MakeFunc(wrapperType, func(args []reflect.Value) []reflect.Value {
		var results []reflect.Value
		if fn.IsVariadic() {
			results = orig.CallSlice(args)
		} else {
			results = orig.Call(args)
		}
		var errVal reflect.Value
		if hasErr {
			errVal = results[1]
			if !errVal.IsNil() {
				return []reflect.Value{reflect.Zero(outType), errVal}
			}
		}
		out := reflect.New(outType).Elem()
		val := results[0]
		fieldIdx := 1
		if includeSelf {
			out.Field(fieldIdx).Set(val)
			fieldIdx++
		}
		for i := range exports {
			out.Field(fieldIdx + i).Set(val)
		}
		if hasErr {
			return []reflect.Value{out, reflect.Zero(errorType)}
		}
		return []reflect.Value{out}
	})

	return wrapper.Interface(), nil
}

func buildGroupedSupply(value any, exports []exportSpec, includeSelf bool) (any, error) {
	if value == nil {
		return nil, fmt.Errorf("supply value must not be nil")
	}
	if _, ok := value.(error); ok {
		return nil, fmt.Errorf("supply value must not be error")
	}
	valueType := reflect.TypeOf(value)
	if err := validateExports(valueType, exports); err != nil {
		return nil, err
	}
	fields := []reflect.StructField{{
		Name:      "Out",
		Type:      reflect.TypeOf(fx.Out{}),
		Anonymous: true,
	}}
	if includeSelf {
		fields = append(fields, reflect.StructField{
			Name: "Self",
			Type: valueType,
		})
	}
	for i, exp := range exports {
		tag := reflect.StructTag("")
		if exp.grouped {
			tag = reflect.StructTag(`group:"` + exp.group + `"`)
		} else if exp.named {
			tag = reflect.StructTag(`name:"` + exp.name + `"`)
		}
		fields = append(fields, reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: exp.typ,
			Tag:  tag,
		})
	}
	outType := reflect.StructOf(fields)

	val := reflect.ValueOf(value)
	fnType := reflect.FuncOf([]reflect.Type{}, []reflect.Type{outType}, false)
	fn := reflect.MakeFunc(fnType, func([]reflect.Value) []reflect.Value {
		out := reflect.New(outType).Elem()
		fieldIdx := 1
		if includeSelf {
			out.Field(fieldIdx).Set(val)
			fieldIdx++
		}
		for i := range exports {
			out.Field(fieldIdx + i).Set(val)
		}
		return []reflect.Value{out}
	})

	return fn.Interface(), nil
}

func validateExports(valueType reflect.Type, exports []exportSpec) error {
	for _, exp := range exports {
		if exp.typ == nil {
			return fmt.Errorf("export type must not be nil")
		}
		if exp.grouped {
			if exp.group == "" {
				return fmt.Errorf("export group must not be empty")
			}
			if exp.named {
				return fmt.Errorf("export cannot be both grouped and named")
			}
		} else if exp.group != "" {
			return fmt.Errorf("export group must be empty for ungrouped export")
		}
		if exp.named && exp.name == "" {
			return fmt.Errorf("export name must not be empty")
		}
		if !valueType.AssignableTo(exp.typ) {
			return fmt.Errorf("%v is not assignable to %v", valueType, exp.typ)
		}
	}
	return nil
}
