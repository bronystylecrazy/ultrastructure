package di

import "go.uber.org/fx"

// Module declares a named module of nodes.
func Module(name string, nodes ...any) Node {
	return moduleNode{name: name, nodes: collectNodes(nodes)}
}

// Options groups nodes without a name.
func Options(nodes ...any) Node {
	return optionsNode{nodes: collectNodes(nodes)}
}

type moduleNode struct {
	name  string
	nodes []Node
}

func (n moduleNode) Build() (fx.Option, error) {
	var opts []fx.Option
	for _, node := range n.nodes {
		opt, err := node.Build()
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}
	return fx.Module(n.name, opts...), nil
}

type optionsNode struct {
	nodes []Node
}

func (n optionsNode) Build() (fx.Option, error) {
	var opts []fx.Option
	for _, node := range n.nodes {
		opt, err := node.Build()
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}
	if len(opts) == 0 {
		return fx.Options(), nil
	}
	if len(opts) == 1 {
		return opts[0], nil
	}
	return fx.Options(opts...), nil
}
