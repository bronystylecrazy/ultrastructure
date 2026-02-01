package di

import (
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/fx"
)

type exportSpec struct {
	typ     reflect.Type
	name    string
	group   string
	grouped bool
	named   bool
}

type bindConfig struct {
	exports                []exportSpec
	includeSelf            bool
	privateSet             bool
	privateValue           bool
	metadata               []any
	pendingNames           []string
	pendingGroups          []string
	autoGroups             []autoGroupRule
	autoGroupIgnores       []autoGroupRule
	ignoreAuto             bool
	autoInjectFields       bool
	ignoreAutoInjectFields bool
	paramTags              []string
	err                    error
}

type bindOptionFunc func(*bindConfig)

func (f bindOptionFunc) applyBind(cfg *bindConfig) { f(cfg) }
func (f bindOptionFunc) applyParam(*paramConfig)   {}

type bothOption []Option

func (b bothOption) applyBind(cfg *bindConfig) {
	for _, opt := range b {
		if opt != nil {
			opt.applyBind(cfg)
		}
	}
}

func (b bothOption) applyParam(cfg *paramConfig) {
	for _, opt := range b {
		if opt != nil {
			opt.applyParam(cfg)
		}
	}
}

// Both groups multiple options together.
func Both(opts ...Option) Option {
	return bothOption(opts)
}

// As exposes the constructor result as type T (non-grouped).
func As[T any](tags ...string) Option {
	return bindOptionFunc(func(cfg *bindConfig) {
		if cfg.err != nil {
			return
		}
		base := exportSpec{
			typ: reflect.TypeOf((*T)(nil)).Elem(),
		}
		if len(tags) > 0 {
			var names []string
			var groups []string
			for _, tag := range tags {
				if tag == "" {
					continue
				}
				n, g := parseAsTag(tag)
				if n == "" && g == "" {
					cfg.err = fmt.Errorf(errAsTagMustIncludeNameOrGroup)
					return
				}
				if n != "" {
					names = append(names, n)
				}
				if g != "" {
					groups = append(groups, g)
				}
			}
			switch {
			case len(names) > 0 && len(groups) > 0:
				for _, name := range names {
					cfg.exports = append(cfg.exports, exportSpec{
						typ:   base.typ,
						name:  name,
						named: true,
					})
				}
				for _, group := range groups {
					cfg.exports = append(cfg.exports, exportSpec{
						typ:     base.typ,
						group:   group,
						grouped: true,
					})
				}
				return
			case len(names) > 0:
				if len(names) > 1 {
					for _, name := range names {
						cfg.exports = append(cfg.exports, exportSpec{
							typ:   base.typ,
							name:  name,
							named: true,
						})
					}
					return
				}
				base.name = names[0]
				base.named = true
			case len(groups) > 0:
				if len(groups) > 1 {
					for _, group := range groups {
						cfg.exports = append(cfg.exports, exportSpec{
							typ:     base.typ,
							group:   group,
							grouped: true,
						})
					}
					return
				}
				base.group = groups[0]
				base.grouped = true
			}
		}
		cfg.exports = append(cfg.exports, base)
	})
}

func parseAsTag(tag string) (string, string) {
	name, _ := extractOptionTagValue(tag, `name:"`)
	group, _ := extractOptionTagValue(tag, `group:"`)
	return name, group
}

func extractOptionTagValue(tag string, key string) (string, bool) {
	idx := strings.Index(tag, key)
	if idx < 0 {
		return "", false
	}
	start := idx + len(key)
	end := strings.Index(tag[start:], `"`)
	if end < 0 {
		return "", false
	}
	end += start
	return tag[start:end], true
}

type nameOption string

func (n nameOption) applyBind(cfg *bindConfig) {
	if cfg.err != nil {
		return
	}
	name := string(n)
	if name == "" {
		cfg.err = fmt.Errorf(errNameEmpty)
		return
	}
	if len(cfg.exports) == 0 {
		cfg.pendingNames = append(cfg.pendingNames, name)
		return
	}
	last := &cfg.exports[len(cfg.exports)-1]
	last.name = name
	last.named = true
	last.group = ""
	last.grouped = false
}

func (n nameOption) applyParam(cfg *paramConfig) {
	if cfg.err != nil {
		return
	}
	name := string(n)
	if name == "" {
		cfg.err = fmt.Errorf(errNameEmpty)
		return
	}
	tag := `name:"` + name + `"`
	cfg.tags = append(cfg.tags, tag)
	cfg.resultTags = append(cfg.resultTags, tag)
}

// Name applies a name tag. For Provide it sets the output name, for Invoke/Decorate it sets the input tag.
func Name(name string) Option {
	return nameOption(name)
}

type groupOption string

func (g groupOption) applyBind(cfg *bindConfig) {
	if cfg.err != nil {
		return
	}
	name := string(g)
	if name == "" {
		cfg.err = fmt.Errorf(errGroupNameEmpty)
		return
	}
	if len(cfg.exports) == 0 {
		cfg.pendingGroups = append(cfg.pendingGroups, name)
		return
	}
	last := &cfg.exports[len(cfg.exports)-1]
	last.group = name
	last.grouped = true
	last.name = ""
	last.named = false
}

func (g groupOption) applyParam(cfg *paramConfig) {
	if cfg.err != nil {
		return
	}
	name := string(g)
	if name == "" {
		cfg.err = fmt.Errorf(errGroupNameEmpty)
		return
	}
	tag := `group:"` + name + `"`
	cfg.tags = append(cfg.tags, tag)
	cfg.resultTags = append(cfg.resultTags, tag)
}

// Group applies a group tag. For Provide it sets the output group, for Invoke/Decorate it sets the input tag.
func Group(name string) Option {
	return groupOption(name)
}

// ToGroup assigns the previous As to a group.
func ToGroup(name string) Option {
	return bothOption{bindOptionFunc(func(cfg *bindConfig) {
		if cfg.err != nil {
			return
		}
		if len(cfg.exports) == 0 {
			cfg.err = fmt.Errorf(errToGroupRequiresAs)
			return
		}
		last := &cfg.exports[len(cfg.exports)-1]
		last.group = name
		last.grouped = true
		last.name = ""
		last.named = false
	})}
}

// Self exposes the concrete type along with any other As* options.
func Self() Option {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.includeSelf = true
	})
}

// AsSelf is an alias for Self.
func AsSelf() Option {
	return Self()
}

// Private hides this constructor from other modules.
func Private() Option {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.privateSet = true
		cfg.privateValue = true
	})
}

// Public clears a previously set Private option.
func Public() Option {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.privateSet = true
		cfg.privateValue = false
	})
}

// AutoGroupIgnore disables auto-grouping for this provider.
func AutoGroupIgnore() Option {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.ignoreAuto = true
	})
}

// AutoInjectIgnore disables auto field injection for this provider.
func AutoInjectIgnore() Option {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.ignoreAutoInjectFields = true
	})
}

func parseBindOptions(opts []any) (bindConfig, []decorateNode, []fx.Option, error) {
	cfg := bindConfig{}
	var decorators []decorateNode
	var extra []fx.Option
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		switch o := opt.(type) {
		case Option:
			o.applyBind(&cfg)
		case decorateNode:
			decorators = append(decorators, o)
		case Node:
			return cfg, nil, nil, fmt.Errorf(errUnsupportedNodeInProvide, opt)
		case fx.Option:
			extra = append(extra, o)
		default:
			return cfg, nil, nil, fmt.Errorf(errUnsupportedOptionType, opt)
		}
		if cfg.err != nil {
			return cfg, nil, nil, cfg.err
		}
	}
	return cfg, decorators, extra, nil
}
