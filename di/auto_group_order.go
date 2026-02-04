package di

import (
	"fmt"
	"math"
	"reflect"
	"sort"
)

type autoGroupOrder struct {
	Index int
}

func applyAutoGroupOrderMetadata(nodes []Node, counter *int, applyAll bool) []Node {
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		switch v := n.(type) {
		case moduleNode:
			v.nodes = applyAutoGroupOrderMetadata(v.nodes, counter, applyAll)
			out = append(out, v)
		case optionsNode:
			v.nodes = applyAutoGroupOrderMetadata(v.nodes, counter, applyAll)
			out = append(out, v)
		case conditionalNode:
			v.nodes = applyAutoGroupOrderMetadata(v.nodes, counter, applyAll)
			out = append(out, v)
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for i, c := range v.cases {
				c.nodes = applyAutoGroupOrderMetadata(c.nodes, counter, applyAll)
				cases[i] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: applyAutoGroupOrderMetadata(v.defaultCase.nodes, counter, applyAll)}
			out = append(out, v)
		case provideNode:
			if applyAll && !hasAutoGroupOrder(v.opts) {
				v.opts = append(append([]any{}, v.opts...), Metadata(autoGroupOrder{Index: *counter}))
			}
			*counter++
			out = append(out, v)
		case supplyNode:
			if applyAll && !hasAutoGroupOrder(v.opts) {
				v.opts = append(append([]any{}, v.opts...), Metadata(autoGroupOrder{Index: *counter}))
			}
			*counter++
			out = append(out, v)
		default:
			out = append(out, n)
		}
	}
	return out
}

func hasAutoGroupOrder(opts []any) bool {
	for _, opt := range opts {
		if m, ok := opt.(metadataAnyOption); ok {
			if _, ok := m.value.(autoGroupOrder); ok {
				return true
			}
		}
	}
	return false
}

func appendAutoGroupOrderDecorators(nodes []Node, priorityGroups map[autoGroupKey]bool) []Node {
	rules := collectAutoGroupRules(nodes)
	if len(rules) == 0 {
		return nodes
	}
	seen := map[autoGroupKey]bool{}
	out := make([]Node, 0, len(nodes)+len(rules))
	out = append(out, nodes...)
	for _, rule := range rules {
		key := autoGroupRuleKey(rule)
		if seen[key] {
			continue
		}
		if len(priorityGroups) == 0 || !priorityGroups[key] {
			continue
		}
		seen[key] = true
		out = append(out, buildAutoGroupOrderDecorator(rule))
	}
	return out
}

func collectPriorityGroups(nodes []Node) (map[autoGroupKey]bool, bool) {
	out := map[autoGroupKey]bool{}
	hasPriority := false
	var walk func([]Node)
	walk = func(items []Node) {
		for _, n := range items {
			switch v := n.(type) {
			case moduleNode:
				walk(v.nodes)
			case optionsNode:
				walk(v.nodes)
			case conditionalNode:
				walk(v.nodes)
			case switchNode:
				for _, c := range v.cases {
					walk(c.nodes)
				}
				walk(v.defaultCase.nodes)
			case provideNode:
				if hasPriorityOpt(v.opts) {
					hasPriority = true
					for _, opt := range v.opts {
						if ag, ok := opt.(autoGroupOption); ok {
							out[autoGroupRuleKey(ag.rule)] = true
						}
					}
				}
			case supplyNode:
				if hasPriorityOpt(v.opts) {
					hasPriority = true
					for _, opt := range v.opts {
						if ag, ok := opt.(autoGroupOption); ok {
							out[autoGroupRuleKey(ag.rule)] = true
						}
					}
				}
			}
		}
	}
	walk(nodes)
	return out, hasPriority
}

func hasPriorityOpt(opts []any) bool {
	for _, opt := range opts {
		if m, ok := opt.(metadataAnyOption); ok {
			if _, ok := m.value.(priorityOrder); ok {
				return true
			}
		}
	}
	return false
}

func buildAutoGroupOrderDecorator(rule autoGroupRule) Node {
	if rule.iface == nil || rule.iface.Kind() != reflect.Interface {
		return errorNode{err: fmt.Errorf(errAutoGroupInterface)}
	}
	sliceType := reflect.SliceOf(rule.iface)
	fnType := reflect.FuncOf([]reflect.Type{sliceType}, []reflect.Type{sliceType}, false)
	fn := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		slice := args[0]
		if slice.Len() <= 1 {
			return []reflect.Value{slice}
		}
		indices := make([]int, slice.Len())
		for i := 0; i < len(indices); i++ {
			indices[i] = i
		}
		sort.SliceStable(indices, func(i, j int) bool {
			left := slice.Index(indices[i]).Interface()
			right := slice.Index(indices[j]).Interface()
			li := autoGroupOrderIndex(left)
			ri := autoGroupOrderIndex(right)
			return li < ri
		})
		out := reflect.MakeSlice(slice.Type(), slice.Len(), slice.Len())
		for i, idx := range indices {
			out.Index(i).Set(slice.Index(idx))
		}
		return []reflect.Value{out}
	})
	return Decorate(fn.Interface(), Group(rule.group))
}

func autoGroupOrderIndex(value any) int {
	raw, ok := ReflectMetadataAny(value)
	if !ok {
		return math.MaxInt
	}
	priority := math.MaxInt
	order := math.MaxInt
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if ord, ok := item.(priorityOrder); ok {
				priority = ord.Index
			}
		}
		for _, item := range v {
			if ord, ok := item.(autoGroupOrder); ok {
				order = ord.Index
			}
		}
	case []string:
		_ = v
	}
	if priority != math.MaxInt {
		if order == math.MaxInt {
			return priority * 1_000_000
		}
		return priority*1_000_000 + order
	}
	return order
}
