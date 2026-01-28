package di

import (
	"fmt"
	"reflect"

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
	exports      []exportSpec
	includeSelf  bool
	privateSet   bool
	privateValue bool
	pendingName  string
	pendingGroup string
	autoGroups   []autoGroupRule
	ignoreAuto   bool
	autoInjectFields bool
	ignoreAutoInjectFields bool
	err          error
}

type Option interface {
	applyBind(*bindConfig)
	applyParam(*paramConfig)
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
func As[T any]() Option {
	return bindOptionFunc(func(cfg *bindConfig) {
		cfg.exports = append(cfg.exports, exportSpec{
			typ:     reflect.TypeOf((*T)(nil)).Elem(),
			grouped: false,
			named:   false,
		})
	})
}

type nameOption string

func (n nameOption) applyBind(cfg *bindConfig) {
	if cfg.err != nil {
		return
	}
	name := string(n)
	if name == "" {
		cfg.err = fmt.Errorf("name must not be empty")
		return
	}
	if len(cfg.exports) == 0 {
		if cfg.pendingName != "" {
			cfg.err = fmt.Errorf("name already set")
			return
		}
		cfg.pendingName = name
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
		cfg.err = fmt.Errorf("name must not be empty")
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
		cfg.err = fmt.Errorf("group name must not be empty")
		return
	}
	if len(cfg.exports) == 0 {
		if cfg.pendingGroup != "" {
			cfg.err = fmt.Errorf("group already set")
			return
		}
		cfg.pendingGroup = name
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
		cfg.err = fmt.Errorf("group name must not be empty")
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
			cfg.err = fmt.Errorf("ToGroup requires a previous As")
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

type paramConfig struct {
	tags       []string
	resultTags []string
	err        error
}

type invokeOptionFunc func(*paramConfig)

func (f invokeOptionFunc) applyBind(*bindConfig) {}
func (f invokeOptionFunc) applyParam(cfg *paramConfig) {
	f(cfg)
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

// InOptional marks the next parameter optional.
func InOptional() Option {
	return InTag(`optional:"true"`)
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
			return cfg, nil, nil, fmt.Errorf("unsupported node type %T inside Provide/Supply", opt)
		case fx.Option:
			extra = append(extra, o)
		default:
			return cfg, nil, nil, fmt.Errorf("unsupported option type %T", opt)
		}
		if cfg.err != nil {
			return cfg, nil, nil, cfg.err
		}
	}
	return cfg, decorators, extra, nil
}
