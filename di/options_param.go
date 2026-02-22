package di

import (
	"fmt"
	"runtime"
	"strings"
)

type paramsOption struct {
	items      []any
	err        error
	sourceFile string
	sourceLine int
}

type skipParamOption struct{}
type variadicParamOption struct {
	items []any
}

func (p paramsOption) applyBind(cfg *bindConfig) {
	if cfg.err != nil {
		return
	}
	cfg.paramsSet = true
	cfg.paramSlots += len(p.items)
	if p.sourceFile != "" && p.sourceLine > 0 {
		cfg.paramsSourceFile = p.sourceFile
		cfg.paramsSourceLine = p.sourceLine
	}
	if p.err != nil {
		cfg.err = p.err
		return
	}
	tags, err := collectParamTags(p.items)
	if err != nil {
		cfg.err = err
		return
	}
	// Collect positional param tags for constructors.
	if len(tags) > 0 {
		cfg.paramTags = append(cfg.paramTags, tags...)
	}
}

func (p paramsOption) applyParam(cfg *paramConfig) {
	if cfg.err != nil {
		return
	}
	cfg.paramsSet = true
	cfg.paramSlots += len(p.items)
	if p.sourceFile != "" && p.sourceLine > 0 {
		cfg.paramsSourceFile = p.sourceFile
		cfg.paramsSourceLine = p.sourceLine
	}
	if p.err != nil {
		cfg.err = p.err
		return
	}
	tags, err := collectParamTags(p.items)
	if err != nil {
		cfg.err = err
		return
	}
	// Collect positional param tags for Invoke/Decorate.
	if len(tags) > 0 {
		cfg.tags = append(cfg.tags, tags...)
	}
}

// Params scopes options to positional parameter tags only.
func Params(items ...any) Option {
	file, line := paramsCallSite()
	p := paramsOption{items: items, sourceFile: file, sourceLine: line}
	if err := validateParamsItems(items); err != nil {
		p.err = err
		return p
	}
	return p
}

func paramsCallSite() (string, int) {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "", 0
	}
	return file, line
}

func collectParamTags(items []any) ([]string, error) {
	var tags []string
	for _, item := range items {
		switch v := item.(type) {
		case nil:
			// Preserve positional placeholders, e.g. Params(nil, Name("x")).
			tags = append(tags, "")
		case string:
			tag := normalizeParamTag(v)
			if tag != "" {
				tags = append(tags, tag)
			}
		case skipParamOption:
			tags = append(tags, "")
		case Option:
			var pc paramConfig
			v.applyParam(&pc)
			if pc.err != nil {
				return nil, pc.err
			}
			if len(pc.tags) > 0 {
				tags = append(tags, pc.tags...)
			}
		}
	}
	return tags, nil
}

func validateParamsItems(items []any) error {
	for _, item := range items {
		switch item.(type) {
		case nil:
			continue
		case string:
			continue
		case skipParamOption:
			continue
		case Option:
			continue
		default:
			// Reject unsupported types early for clearer errors.
			return fmt.Errorf(errOptionalParamTagType, item)
		}
	}
	return nil
}

func normalizeParamTag(tag string) string {
	if strings.TrimSpace(tag) == "" {
		return `optional:"true"`
	}
	if strings.Contains(tag, `groups:"`) && !strings.Contains(tag, `group:"`) {
		return strings.Replace(tag, `groups:"`, `group:"`, 1)
	}
	return tag
}

type paramConfig struct {
	tags             []string
	resultTags       []string
	paramSlots       int
	variadicShorthand bool
	paramsSet        bool
	paramsSourceFile string
	paramsSourceLine int
	err              error
}

type invokeOptionFunc func(*paramConfig)

func (f invokeOptionFunc) applyBind(*bindConfig) {}
func (f invokeOptionFunc) applyParam(cfg *paramConfig) {
	f(cfg)
}

// applyParamOptions applies param options and returns the first error encountered.
func applyParamOptions(opts []Option, cfg *paramConfig) error {
	for _, opt := range opts {
		if opt != nil {
			opt.applyParam(cfg)
		}
		if cfg.err != nil {
			return cfg.err
		}
	}
	return nil
}

// InTag appends a param tag in positional order.
func InTag(tag string) Option {
	return invokeOptionFunc(func(cfg *paramConfig) {
		cfg.tags = append(cfg.tags, tag)
	})
}

// InGroup tags the next parameter for a group.
func InGroup(name string) Option {
	return InTag(`group:"` + name + `"`)
}

// Optional marks the next parameter optional.
func Optional() Option {
	return InTag(`optional:"true"`)
}

func (skipParamOption) applyBind(*bindConfig) {}
func (skipParamOption) applyParam(cfg *paramConfig) {
	cfg.tags = append(cfg.tags, "")
}

// Skip reserves a positional parameter slot in Params without adding a tag.
func Skip() Option {
	return skipParamOption{}
}

func (v variadicParamOption) applyBind(cfg *bindConfig) {
	if cfg.err != nil {
		return
	}
	tags, err := collectParamTags(v.items)
	if err != nil {
		cfg.err = err
		return
	}
	if len(tags) != 1 {
		cfg.err = fmt.Errorf(errVariadicParamTagSingle, len(tags))
		return
	}
	if cfg.paramsSet && !cfg.variadicShorthand && cfg.paramSlots > 0 {
		cfg.err = fmt.Errorf(errVariadicWithParams)
		return
	}
	cfg.paramsSet = true
	cfg.variadicShorthand = true
	cfg.paramTags = append(cfg.paramTags, tags[0])
}
func (v variadicParamOption) applyParam(cfg *paramConfig) {
	if cfg.err != nil {
		return
	}
	tags, err := collectParamTags(v.items)
	if err != nil {
		cfg.err = err
		return
	}
	if len(tags) != 1 {
		cfg.err = fmt.Errorf(errVariadicParamTagSingle, len(tags))
		return
	}
	if cfg.paramsSet && !cfg.variadicShorthand && cfg.paramSlots > 0 {
		cfg.err = fmt.Errorf(errVariadicWithParams)
		return
	}
	cfg.paramsSet = true
	cfg.variadicShorthand = true
	cfg.tags = append(cfg.tags, tags[0])
}

// Variadic tags the last variadic parameter slot with a single tag.
// Used directly, it is shorthand for padding preceding Params slots with empty tags.
func Variadic(items ...any) Option {
	return variadicParamOption{items: items}
}

// VariadicGroup tags the last variadic parameter slot as a group dependency.
func VariadicGroup(name string) Option {
	return Variadic(Group(name))
}
