package di

import (
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/fx"
)

// AutoGroup registers an interface to be auto-grouped within the scope.
// Default group name is lowercased interface name unless provided.
func AutoGroup[T any](group ...string) Node {
	iface := reflect.TypeOf((*T)(nil)).Elem()
	if iface.Kind() != reflect.Interface {
		return errorNode{err: fmt.Errorf("AutoGroup expects an interface type")}
	}
	name := ""
	if len(group) > 0 {
		name = group[0]
	}
	if name == "" {
		name = strings.ToLower(iface.Name())
	}
	return autoGroupNode{rule: autoGroupRule{iface: iface, group: name}}
}

type autoGroupRule struct {
	iface  reflect.Type
	group  string
	filter func(reflect.Type) bool
	asSelf bool
}

type autoGroupNode struct {
	rule autoGroupRule
}

func (n autoGroupNode) Build() (fx.Option, error) {
	return fx.Options(), nil
}

type autoGroupOption struct {
	rule autoGroupRule
}

func (o autoGroupOption) applyBind(cfg *bindConfig) {
	if o.rule.filter != nil || o.rule.asSelf {
		applyAutoGroupRuleOverrides(cfg, o.rule)
		return
	}
	cfg.autoGroups = append(cfg.autoGroups, o.rule)
}

func (o autoGroupOption) applyParam(*paramConfig) {}

type autoGroupApplier interface {
	withAutoGroups([]autoGroupRule) Node
}

func applyAutoGroups(nodes []Node, inherited []autoGroupRule) []Node {
	local := append([]autoGroupRule{}, inherited...)
	for _, n := range nodes {
		if g, ok := n.(autoGroupNode); ok {
			local = append(local, g.rule)
		}
	}
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		switch v := n.(type) {
		case moduleNode:
			v.nodes = applyAutoGroups(v.nodes, local)
			out[i] = v
		case optionsNode:
			v.nodes = applyAutoGroups(v.nodes, local)
			out[i] = v
		case conditionalNode:
			v.nodes = applyAutoGroups(v.nodes, local)
			out[i] = v
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for idx, c := range v.cases {
				c.nodes = applyAutoGroups(c.nodes, local)
				cases[idx] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: applyAutoGroups(v.defaultCase.nodes, local)}
			out[i] = v
		default:
			if applier, ok := n.(autoGroupApplier); ok {
				out[i] = applier.withAutoGroups(local)
				continue
			}
			out[i] = n
		}
	}
	return out
}

func appendAutoGroupOptions(opts []any, rules []autoGroupRule) []any {
	if len(rules) == 0 {
		return opts
	}
	for _, rule := range rules {
		if hasAutoGroupOption(opts, rule) {
			continue
		}
		opts = append(opts, autoGroupOption{rule: rule})
	}
	return opts
}

func hasAutoGroupOption(opts []any, rule autoGroupRule) bool {
	for _, opt := range opts {
		ag, ok := opt.(autoGroupOption)
		if !ok {
			continue
		}
		if ag.rule.iface == rule.iface && ag.rule.group == rule.group {
			return true
		}
	}
	return false
}

func applyAutoGroupRuleOverrides(cfg *bindConfig, override autoGroupRule) {
	for i := range cfg.autoGroups {
		rule := cfg.autoGroups[i]
		if rule.iface != override.iface || rule.group != override.group {
			continue
		}
		if override.filter != nil {
			rule.filter = override.filter
		}
		if override.asSelf {
			rule.asSelf = true
		}
		cfg.autoGroups[i] = rule
		return
	}
	cfg.autoGroups = append(cfg.autoGroups, override)
}

// AutoGroupFilter narrows auto grouping to types that pass the predicate.
func AutoGroupFilter(fn func(reflect.Type) bool) Option {
	return autoGroupOption{rule: autoGroupRule{filter: fn}}
}

// AutoGroupAsSelf ensures the concrete type is provided alongside auto-grouping.
func AutoGroupAsSelf() Option {
	return autoGroupOption{rule: autoGroupRule{asSelf: true}}
}

func (n provideNode) withAutoGroups(rules []autoGroupRule) Node {
	opts := appendAutoGroupOptions(append([]any{}, n.opts...), rules)
	return provideNode{constructor: n.constructor, opts: opts}
}

func (n supplyNode) withAutoGroups(rules []autoGroupRule) Node {
	opts := appendAutoGroupOptions(append([]any{}, n.opts...), rules)
	return supplyNode{value: n.value, opts: opts}
}

func (n configNode[T]) withAutoGroups(rules []autoGroupRule) Node {
	opts := appendAutoGroupOptions(append([]any{}, n.opts...), rules)
	return configNode[T]{pathOrKey: n.pathOrKey, opts: opts, scope: n.scope}
}

func (n configBindNode[T]) withAutoGroups(rules []autoGroupRule) Node {
	opts := appendAutoGroupOptions(append([]any{}, n.opts...), rules)
	return configBindNode[T]{key: n.key, opts: opts, scope: n.scope}
}
