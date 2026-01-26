package di

import (
	"errors"
	"fmt"
	"reflect"

	"go.uber.org/fx"
)

// If conditionally includes nodes based on a boolean.
func If(cond bool, nodes ...any) Node {
	return conditionalNode{mode: condIf, cond: cond, nodes: collectNodes(nodes)}
}

// When conditionally includes nodes based on a function.
// The function may be func() bool or func(T) bool, where T is resolved from config sources.
func When(fn any, nodes ...any) Node {
	return conditionalNode{mode: condWhen, when: fn, nodes: collectNodes(nodes)}
}

type condMode int

const (
	condIf condMode = iota
	condWhen
)

type conditionalNode struct {
	mode      condMode
	cond      bool
	when      any
	nodes     []Node
	evaluated bool
	result    bool
	resolver  func(reflect.Type) (any, error)
}

func (n *conditionalNode) eval() (bool, error) {
	if n.evaluated {
		return n.result, nil
	}
	switch n.mode {
	case condIf:
		n.result = n.cond
	case condWhen:
		if n.when == nil {
			return false, fmt.Errorf("when function must not be nil")
		}
		ok, err := evalWhen(n.when, n.resolver)
		if err != nil {
			return false, err
		}
		n.result = ok
	default:
		return false, fmt.Errorf("unknown condition mode")
	}
	n.evaluated = true
	return n.result, nil
}

func (n conditionalNode) Build() (fx.Option, error) {
	ok, err := (&n).eval()
	if err != nil {
		return nil, err
	}
	if !ok {
		return fx.Options(), nil
	}
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

func evalWhen(fn any, resolver func(reflect.Type) (any, error)) (bool, error) {
	fnType := reflect.TypeOf(fn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return false, fmt.Errorf("when must be a function")
	}
	if fnType.NumOut() != 1 || fnType.Out(0).Kind() != reflect.Bool {
		return false, fmt.Errorf("when must return bool")
	}
	if fnType.NumIn() > 8 {
		return false, fmt.Errorf("when must accept at most 8 parameters")
	}
	var args []reflect.Value
	if fnType.NumIn() > 0 {
		if resolver == nil {
			return false, fmt.Errorf("when requires config values but no config source available")
		}
		for i := 0; i < fnType.NumIn(); i++ {
			paramType := fnType.In(i)
			val, err := resolver(paramType)
			if err != nil {
				var notFound configTypeNotFoundError
				if errors.As(err, &notFound) {
					return false, fmt.Errorf("when parameter %s is not a config type", paramType)
				}
				return false, err
			}
			if val == nil {
				return false, fmt.Errorf("when could not resolve %s", paramType)
			}
			args = append(args, reflect.ValueOf(val))
		}
	}
	out := reflect.ValueOf(fn).Call(args)
	return out[0].Bool(), nil
}
