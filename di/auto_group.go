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
		return errorNode{err: fmt.Errorf(errAutoGroupInterface)}
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
	withAutoGroups(autoGroupSet) Node
}

type autoGroupSet struct {
	rules []autoGroupRule
	opts  []any
}

type autoGroupKey struct {
	iface reflect.Type
	group string
}

func autoGroupRuleKey(rule autoGroupRule) autoGroupKey {
	return autoGroupKey{iface: rule.iface, group: rule.group}
}

func buildAutoGroupSet(inherited []autoGroupRule, nodes []Node) autoGroupSet {
	local := append([]autoGroupRule{}, inherited...)
	// Collect AutoGroup rules from the current scope (including Options/conditions).
	local = append(local, collectAutoGroupRules(nodes)...)
	return autoGroupSet{
		rules: local,
		opts:  buildAutoGroupOptions(local),
	}
}

func buildAutoGroupOptions(rules []autoGroupRule) []any {
	if len(rules) == 0 {
		return nil
	}
	opts := make([]any, len(rules))
	for i, rule := range rules {
		opts[i] = autoGroupOption{rule: rule}
	}
	return opts
}

func applyAutoGroups(nodes []Node, inherited []autoGroupRule) []Node {
	localSet := buildAutoGroupSet(inherited, nodes)
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		switch v := n.(type) {
		case moduleNode:
			v.nodes = applyAutoGroups(v.nodes, localSet.rules)
			out[i] = v
		case optionsNode:
			v.nodes = applyAutoGroups(v.nodes, localSet.rules)
			out[i] = v
		case conditionalNode:
			v.nodes = applyAutoGroups(v.nodes, localSet.rules)
			out[i] = v
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for idx, c := range v.cases {
				c.nodes = applyAutoGroups(c.nodes, localSet.rules)
				cases[idx] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: applyAutoGroups(v.defaultCase.nodes, localSet.rules)}
			out[i] = v
		default:
			if applier, ok := n.(autoGroupApplier); ok {
				out[i] = applier.withAutoGroups(localSet)
				continue
			}
			out[i] = n
		}
	}
	return out
}

func collectAutoGroupRules(nodes []Node) []autoGroupRule {
	var rules []autoGroupRule
	for _, n := range nodes {
		switch v := n.(type) {
		case autoGroupNode:
			rules = append(rules, v.rule)
		case optionsNode:
			rules = append(rules, collectAutoGroupRules(v.nodes)...)
		case conditionalNode:
			rules = append(rules, collectAutoGroupRules(v.nodes)...)
		case switchNode:
			for _, c := range v.cases {
				rules = append(rules, collectAutoGroupRules(c.nodes)...)
			}
			rules = append(rules, collectAutoGroupRules(v.defaultCase.nodes)...)
		}
	}
	return rules
}

func addAutoGroupOptions(opts []any, set autoGroupSet) ([]any, bool) {
	if len(set.rules) == 0 {
		return opts, false
	}
	if len(opts) == 0 {
		return set.opts, true
	}
	existing := map[autoGroupKey]bool{}
	for _, opt := range opts {
		if ag, ok := opt.(autoGroupOption); ok {
			existing[autoGroupRuleKey(ag.rule)] = true
		}
	}
	added := 0
	for _, rule := range set.rules {
		if existing[autoGroupRuleKey(rule)] {
			continue
		}
		added++
	}
	if added == 0 {
		return opts, false
	}
	out := make([]any, 0, len(opts)+added)
	out = append(out, opts...)
	for _, rule := range set.rules {
		if existing[autoGroupRuleKey(rule)] {
			continue
		}
		out = append(out, autoGroupOption{rule: rule})
	}
	return out, true
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

// AutoGroupIgnoreType ignores auto-grouping for the given interface.
// If a group is provided, it targets that group; otherwise it ignores all groups for the interface.
func AutoGroupIgnoreType[T any](group ...string) Option {
	iface := reflect.TypeOf((*T)(nil)).Elem()
	if iface.Kind() != reflect.Interface {
		return errorOption{err: fmt.Errorf(errAutoGroupIgnoreTypeInterface)}
	}
	name := ""
	if len(group) > 0 {
		name = group[0]
	}
	return autoGroupIgnoreOption{rule: autoGroupRule{iface: iface, group: name}}
}

type autoGroupIgnoreOption struct {
	rule autoGroupRule
}

func (o autoGroupIgnoreOption) applyBind(cfg *bindConfig) {
	cfg.autoGroupIgnores = append(cfg.autoGroupIgnores, o.rule)
}

func (o autoGroupIgnoreOption) applyParam(*paramConfig) {}

func (n provideNode) withAutoGroups(set autoGroupSet) Node {
	opts, changed := addAutoGroupOptions(n.opts, set)
	if !changed {
		return n
	}
	return provideNode{
		constructor:       n.constructor,
		opts:              opts,
		sourceFile:        n.sourceFile,
		sourceLine:        n.sourceLine,
		paramTagsOverride: n.paramTagsOverride,
	}
}

func (n supplyNode) withAutoGroups(set autoGroupSet) Node {
	opts, changed := addAutoGroupOptions(n.opts, set)
	if !changed {
		return n
	}
	return supplyNode{value: n.value, opts: opts}
}

type errorOption struct {
	err error
}

func (o errorOption) applyBind(cfg *bindConfig) {
	if cfg.err == nil {
		cfg.err = o.err
	}
}

func (o errorOption) applyParam(cfg *paramConfig) {
	if cfg.err == nil {
		cfg.err = o.err
	}
}
