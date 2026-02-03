package di

import (
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

// Case defines a boolean case for Switch.
func Case(cond bool, nodes ...any) caseNode {
	return caseNode{mode: condIf, cond: cond, nodes: collectNodes(nodes)}
}

// WhenCase defines a predicate case for Switch.
// The function may be func() bool or func(T) bool, where T is resolved from config sources.
func WhenCase(fn any, nodes ...any) caseNode {
	return caseNode{mode: condWhen, when: fn, nodes: collectNodes(nodes)}
}

// Switch selects the first matching Case, or Default if none match.
func Switch(items ...any) Node {
	var s switchNode
	for _, it := range items {
		switch v := it.(type) {
		case caseNode:
			s.cases = append(s.cases, v)
		case *caseNode:
			if v != nil {
				s.cases = append(s.cases, *v)
			}
		case switchDefaultNode:
			if s.hasDefault {
				return errorNode{err: fmt.Errorf(errSwitchDefaultOnlyOnce)}
			}
			s.defaultCase = v
			s.hasDefault = true
		case *switchDefaultNode:
			if v == nil {
				continue
			}
			if s.hasDefault {
				return errorNode{err: fmt.Errorf(errSwitchDefaultOnlyOnce)}
			}
			s.defaultCase = *v
			s.hasDefault = true
		default:
			return errorNode{err: fmt.Errorf(errSwitchUnsupportedNode, it)}
		}
	}
	return s
}

type caseNode struct {
	mode     condMode
	cond     bool
	when     any
	nodes    []Node
	resolver func(reflect.Type) (any, error)
}

// DefaultCase defines the default branch for Switch.
func DefaultCase(nodes ...any) switchDefaultNode {
	return switchDefaultNode{nodes: collectNodes(nodes)}
}

type switchDefaultNode struct {
	nodes []Node
}

type switchNode struct {
	cases       []caseNode
	defaultCase switchDefaultNode
	hasDefault  bool
	resolved    bool
	selected    []Node
}

func (n switchNode) Build() (fx.Option, error) {
	nodes, err := n.selectNodes()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return fx.Options(), nil
	}
	var opts []fx.Option
	for _, node := range nodes {
		opt, err := node.Build()
		if err != nil {
			return nil, err
		}
		opts = append(opts, opt)
	}
	return packOptions(opts), nil
}

func (n switchNode) selectNodes() ([]Node, error) {
	if n.resolved {
		// Avoid re-evaluating cases after resolution.
		return n.selected, nil
	}
	for _, c := range n.cases {
		ok, err := c.eval()
		if err != nil {
			return nil, err
		}
		if ok {
			return c.nodes, nil
		}
	}
	if n.hasDefault {
		return n.defaultCase.nodes, nil
	}
	return nil, nil
}

func (n caseNode) eval() (bool, error) {
	switch n.mode {
	case condIf:
		return n.cond, nil
	case condWhen:
		if n.when == nil {
			return false, fmt.Errorf(errWhenFunctionNil)
		}
		return evalWhen(n.when, n.resolver)
	default:
		return false, fmt.Errorf(errUnknownConditionMode)
	}
}
