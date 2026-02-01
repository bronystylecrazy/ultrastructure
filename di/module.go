package di

import (
	"fmt"

	"go.uber.org/fx"
)

// Module declares a named module of nodes.
func Module(name string, nodes ...any) Node {
	return moduleNode{name: name, nodes: collectNodes(nodes)}
}

// Options groups nodes without a name.
func Options(nodes ...any) NodeOption {
	return optionsNode{items: nodes, nodes: collectNodes(nodes)}
}

type moduleNode struct {
	name  string
	nodes []Node
}

func (n moduleNode) Build() (fx.Option, error) {
	var opts []fx.Option
	for _, node := range n.nodes {
		if _, ok := node.(decorateNode); ok {
			// Decorators are composed globally at the app level.
			continue
		}
		opt, err := node.Build()
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}
	return fx.Module(n.name, packOptions(opts)), nil
}

type optionsNode struct {
	items []any
	nodes []Node
}

func (n optionsNode) Build() (fx.Option, error) {
	var opts []fx.Option
	for _, node := range n.nodes {
		if _, ok := node.(decorateNode); ok {
			// Decorators are composed globally at the app level.
			continue
		}
		opt, err := node.Build()
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}
	return packOptions(opts), nil
}

func (n optionsNode) applyBind(cfg *bindConfig) {
	for _, item := range n.items {
		switch v := item.(type) {
		case nil:
			continue
		case Option:
			v.applyBind(cfg)
		default:
			cfg.err = fmt.Errorf(errUnsupportedOptionType, item)
		}
		if cfg.err != nil {
			return
		}
	}
}

func (n optionsNode) applyParam(cfg *paramConfig) {
	for _, item := range n.items {
		switch v := item.(type) {
		case nil:
			continue
		case Option:
			v.applyParam(cfg)
		default:
			cfg.err = fmt.Errorf(errUnsupportedOptionType, item)
		}
		if cfg.err != nil {
			return
		}
	}
}
