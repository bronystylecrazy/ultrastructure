package us

import (
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

type provideConfig struct {
	exports      []exportSpec
	includeSelf  bool
	privateSet   bool
	privateValue bool
}

type ProvideOption interface {
	apply(*provideConfig)
}

type provideOptionFunc func(*provideConfig)

func (f provideOptionFunc) apply(cfg *provideConfig) {
	f(cfg)
}

// Provide wraps fx.Provide + fx.Annotate to keep call sites concise.
func Provide(constructor any, opts ...ProvideOption) fx.Option {
	var cfg provideConfig
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&cfg)
		}
	}
	if len(cfg.exports) == 0 && !cfg.includeSelf {
		if cfg.privateSet && cfg.privateValue {
			return fx.Provide(constructor, fx.Private)
		}
		return fx.Provide(constructor)
	}
	wrapped, err := buildGroupedConstructor(constructor, cfg.exports, cfg.includeSelf)
	if err != nil {
		return fx.Error(err)
	}
	if cfg.privateSet && cfg.privateValue {
		return fx.Provide(wrapped, fx.Private)
	}
	return fx.Provide(wrapped)
}

// AsSelf exposes the concrete type along with any other As* options.
func AsSelf() ProvideOption {
	return provideOptionFunc(func(cfg *provideConfig) {
		cfg.includeSelf = true
	})
}

// Private marks the provided constructor/supply as private to the module.
func Private() ProvideOption {
	return provideOptionFunc(func(cfg *provideConfig) {
		cfg.privateSet = true
		cfg.privateValue = true
	})
}

// Public clears a previously set Private option.
func Public() ProvideOption {
	return provideOptionFunc(func(cfg *provideConfig) {
		cfg.privateSet = true
		cfg.privateValue = false
	})
}

// Module wraps fx.Module to group options under a named module.
func Module(name string, opts ...fx.Option) fx.Option {
	return fx.Module(name, opts...)
}

// Options wraps fx.Options to group multiple options without a module name.
func Options(opts ...fx.Option) fx.Option {
	return fx.Options(opts...)
}

// Supply wraps fx.Supply and supports ProvideOption annotations for a single value.
func Supply(values ...any) fx.Option {
	if len(values) == 0 {
		return fx.Supply()
	}

	firstOpt := -1
	for i, v := range values {
		if _, ok := v.(ProvideOption); ok {
			firstOpt = i
			break
		}
	}

	if firstOpt == -1 {
		return fx.Supply(values...)
	}

	if firstOpt != 1 {
		return fx.Error(fmt.Errorf("supply with options requires exactly one value followed by options"))
	}

	val := values[0]
	var cfg provideConfig
	for _, v := range values[1:] {
		opt, ok := v.(ProvideOption)
		if !ok {
			return fx.Error(fmt.Errorf("supply with options accepts only ProvideOption after the value"))
		}
		opt.apply(&cfg)
	}
	if len(cfg.exports) == 0 && !cfg.includeSelf {
		if cfg.privateSet && cfg.privateValue {
			return fx.Supply(val, fx.Private)
		}
		return fx.Supply(val)
	}
	wrapped, err := buildGroupedSupply(val, cfg.exports, cfg.includeSelf)
	if err != nil {
		return fx.Error(err)
	}
	if cfg.privateSet && cfg.privateValue {
		return fx.Provide(wrapped, fx.Private)
	}
	return fx.Provide(wrapped)
}

// Replace wraps fx.Replace.
func Replace(values ...any) fx.Option {
	if len(values) == 0 {
		return fx.Replace()
	}

	firstOpt := -1
	for i, v := range values {
		if _, ok := v.(ProvideOption); ok {
			firstOpt = i
			break
		}
	}

	if firstOpt == -1 {
		return fx.Replace(values...)
	}

	if firstOpt != 1 {
		return fx.Error(fmt.Errorf("replace with options requires exactly one value followed by options"))
	}

	val := values[0]
	var cfg provideConfig
	for _, v := range values[1:] {
		opt, ok := v.(ProvideOption)
		if !ok {
			return fx.Error(fmt.Errorf("replace with options accepts only ProvideOption after the value"))
		}
		opt.apply(&cfg)
	}

	if len(cfg.exports) == 0 && !cfg.includeSelf {
		return fx.Replace(val)
	}

	for _, exp := range cfg.exports {
		if exp.grouped {
			return fx.Error(fmt.Errorf("replace does not support groups"))
		}
		if exp.named {
			return fx.Error(fmt.Errorf("replace does not support named exports"))
		}
		if exp.typ.Kind() != reflect.Interface {
			return fx.Error(fmt.Errorf("replace AsType requires an interface type, got %v", exp.typ))
		}
	}

	var anns []fx.Annotation
	if cfg.includeSelf {
		anns = append(anns, fx.As(fx.Self()))
	}
	for _, exp := range cfg.exports {
		anns = append(anns, fx.As(reflect.New(exp.typ).Interface()))
	}

	if len(anns) == 0 {
		return fx.Replace(val)
	}
	return fx.Replace(fx.Annotate(val, anns...))
}

type paramConfig struct {
	paramTags   []string
	annotations []fx.Annotation
}

type InvokeOption interface {
	apply(*paramConfig)
}

type invokeOptionFunc func(*paramConfig)

func (f invokeOptionFunc) apply(cfg *paramConfig) {
	f(cfg)
}

// Invoke wraps fx.Invoke + fx.Annotate to keep call sites concise.
func Invoke(function any, opts ...InvokeOption) fx.Option {
	var cfg paramConfig
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&cfg)
		}
	}
	if len(cfg.paramTags) > 0 {
		cfg.annotations = append(cfg.annotations, fx.ParamTags(cfg.paramTags...))
	}
	if len(cfg.annotations) == 0 {
		return fx.Invoke(function)
	}
	return fx.Invoke(fx.Annotate(function, cfg.annotations...))
}

// InTag appends a param tag for Invoke in positional order.
func InTag(tag string) InvokeOption {
	return invokeOptionFunc(func(cfg *paramConfig) {
		cfg.paramTags = append(cfg.paramTags, tag)
	})
}

// InOptional marks the next Invoke parameter as optional.
func InOptional() InvokeOption {
	return InTag(`optional:"true"`)
}

// InName tags the next Invoke parameter with a name.
func InName(name string) InvokeOption {
	return InTag(`name:"` + name + `"`)
}

// InvokeAnn adds a raw fx.Annotation to Invoke.
func InvokeAnn(ann fx.Annotation) InvokeOption {
	return invokeOptionFunc(func(cfg *paramConfig) {
		if ann != nil {
			cfg.annotations = append(cfg.annotations, ann)
		}
	})
}

type DecorateOption interface {
	apply(*paramConfig)
}

type decorateOptionFunc func(*paramConfig)

func (f decorateOptionFunc) apply(cfg *paramConfig) {
	f(cfg)
}

// Decorate wraps fx.Decorate + fx.Annotate to keep call sites concise.
func Decorate(function any, opts ...DecorateOption) fx.Option {
	var cfg paramConfig
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&cfg)
		}
	}
	if len(cfg.paramTags) > 0 {
		cfg.annotations = append(cfg.annotations, fx.ParamTags(cfg.paramTags...))
	}
	if len(cfg.annotations) == 0 {
		return fx.Decorate(function)
	}
	return fx.Decorate(fx.Annotate(function, cfg.annotations...))
}

// DecorateTag appends a param tag for Decorate in positional order.
func DecorateTag(tag string) DecorateOption {
	return decorateOptionFunc(func(cfg *paramConfig) {
		cfg.paramTags = append(cfg.paramTags, tag)
	})
}

// DecorateOptional marks the next Decorate parameter as optional.
func DecorateOptional() DecorateOption {
	return DecorateTag(`optional:"true"`)
}

// DecorateName tags the next Decorate parameter with a name.
func DecorateName(name string) DecorateOption {
	return DecorateTag(`name:"` + name + `"`)
}

// DecorateAnn adds a raw fx.Annotation to Decorate.
func DecorateAnn(ann fx.Annotation) DecorateOption {
	return decorateOptionFunc(func(cfg *paramConfig) {
		if ann != nil {
			cfg.annotations = append(cfg.annotations, ann)
		}
	})
}

type exportSpec struct {
	typ     reflect.Type
	group   string
	grouped bool
	name    string
	named   bool
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

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
