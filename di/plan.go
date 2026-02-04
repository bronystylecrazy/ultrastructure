package di

import (
	"fmt"
	"reflect"
	"strings"
)

// Plan builds the app graph and returns a readable plan.
func Plan(nodes ...any) (string, error) {
	return App(nodes...).Plan()
}

// Plan builds the app graph and returns a readable plan.
func (a *appNode) Plan() (string, error) {
	nextID := 0
	nextScopeID := 1
	// Build a resolved node list similar to Build(), but only for planning.
	resolver := buildConfigResolver(a.nodes)
	nodes := attachConfigResolvers(a.nodes, resolver)
	nodes = applyAutoGroups(nodes, nil)
	nodes, err := applyReplacements(nodes, nil, &nextID, &nextScopeID, []int{0}, nil)
	if err != nil {
		return "", err
	}
	priorityGroups, hasPriority := collectPriorityGroups(nodes)
	if hasPriority {
		orderCounter := 0
		nodes = applyAutoGroupOrderMetadata(nodes, &orderCounter, true)
		if len(priorityGroups) > 0 {
			nodes = appendAutoGroupOrderDecorators(nodes, priorityGroups)
		}
	}
	var b strings.Builder
	for i, n := range nodes {
		writePlanNode(&b, n, "", i == len(nodes)-1)
	}
	return b.String(), nil
}

func writePlanNode(b *strings.Builder, n Node, prefix string, isLast bool) {
	branch := "|-- "
	nextPrefix := prefix + "|   "
	if isLast {
		branch = "`-- "
		nextPrefix = prefix + "    "
	}

	label, children := planNodeLabel(n)
	fmt.Fprintf(b, "%s%s%s\n", prefix, branch, label)
	for i, child := range children {
		writePlanNode(b, child, nextPrefix, i == len(children)-1)
	}
}

func planNodeLabel(n Node) (string, []Node) {
	switch v := n.(type) {
	case moduleNode:
		return fmt.Sprintf("Module %q", v.name), v.nodes
	case optionsNode:
		return "Options", v.nodes
	case switchNode:
		selected, err := v.selectNodes()
		if err != nil {
			return fmt.Sprintf("Switch <error: %v>", err), nil
		}
		if len(selected) == 0 {
			return "Switch (no match)", nil
		}
		return "Switch", selected
	case conditionalNode:
		ok, err := v.eval()
		if err != nil {
			return fmt.Sprintf("If/When <error: %v>", err), nil
		}
		if !ok {
			return "If/When (skipped)", nil
		}
		label := "If"
		if v.mode == condWhen {
			label = "When"
		}
		return label, v.nodes
	case provideNode:
		desc, err := describeProvide(v)
		if err != nil {
			return fmt.Sprintf("Provide <error: %v>", err), nil
		}
		return fmt.Sprintf("Provide %s", desc), nil
	case supplyNode:
		desc, err := describeSupply(v)
		if err != nil {
			return fmt.Sprintf("Supply <error: %v>", err), nil
		}
		return fmt.Sprintf("Supply %s", desc), nil
	case replaceNode:
		desc, err := describeReplace(v)
		if err != nil {
			return fmt.Sprintf("Replace <error: %v>", err), nil
		}
		return fmt.Sprintf("Replace %s", desc), nil
	case defaultNode:
		desc, err := describeDefault(v)
		if err != nil {
			return fmt.Sprintf("Default <error: %v>", err), nil
		}
		return fmt.Sprintf("Default %s", desc), nil
	case invokeNode:
		desc, err := describeInvoke(v)
		if err != nil {
			return fmt.Sprintf("Invoke <error: %v>", err), nil
		}
		return fmt.Sprintf("Invoke %s", desc), nil
	case populateNode:
		desc, err := describePopulate(v)
		if err != nil {
			return fmt.Sprintf("Populate <error: %v>", err), nil
		}
		return fmt.Sprintf("Populate %s", desc), nil
	case autoGroupNode:
		return fmt.Sprintf("AutoGroup %s -> %q", v.rule.iface.String(), v.rule.group), nil
	case decorateNode:
		desc, err := describeDecorate(v)
		if err != nil {
			return fmt.Sprintf("Decorate <error: %v>", err), nil
		}
		return fmt.Sprintf("Decorate %s", desc), nil
	case fxOptionNode:
		return "FxOption", nil
	case lifecycleNode:
		if v.kind == lifecycleStart {
			return "OnStart", nil
		}
		return "OnStop", nil
	case errorNode:
		return fmt.Sprintf("Error %v", v.err), nil
	default:
		return fmt.Sprintf("%T", n), nil
	}
}

func describeProvide(n provideNode) (string, error) {
	cfg, _, _, err := parseBindOptions(n.opts)
	if err != nil {
		return "", err
	}
	_, tagSets, err := buildProvideSpec(cfg, n.constructor, nil)
	if err != nil {
		return "", err
	}
	typ, err := constructorResultType(n.constructor)
	if err != nil {
		return "", err
	}
	return formatTypeAndTags(typ, tagSets), nil
}

func describeSupply(n supplyNode) (string, error) {
	cfg, _, _, err := parseBindOptions(n.opts)
	if err != nil {
		return "", err
	}
	_, tagSets, err := buildProvideSpec(cfg, nil, n.value)
	if err != nil {
		return "", err
	}
	typ := reflect.TypeOf(n.value)
	if typ == nil {
		return "", fmt.Errorf(errValueMustNotBeNil)
	}
	return formatTypeAndTags(typ, tagSets), nil
}

func describeReplace(n replaceNode) (string, error) {
	spec, err := buildReplaceSpec(n, 0)
	if err != nil {
		return "", err
	}
	typ, err := replacementBaseType(spec.node)
	if err != nil {
		return "", err
	}
	mode := "all"
	switch spec.mode {
	case replaceBefore:
		mode = "before"
	case replaceAfter:
		mode = "after"
	}
	return fmt.Sprintf("%s %s", formatTypeAndTags(typ, spec.tagSets), mode), nil
}

func describeDefault(n defaultNode) (string, error) {
	spec, err := buildDefaultSpec(n, 0)
	if err != nil {
		return "", err
	}
	typ, err := replacementBaseType(spec.node)
	if err != nil {
		return "", err
	}
	return formatTypeAndTags(typ, spec.tagSets), nil
}

func describeInvoke(n invokeNode) (string, error) {
	fnType := reflect.TypeOf(n.function)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return "", fmt.Errorf(errInvokeMustBeFunction)
	}
	var cfg paramConfig
	if err := applyParamOptions(n.opts, &cfg); err != nil {
		return "", err
	}
	if len(cfg.tags) == 0 {
		return fnType.String(), nil
	}
	return fmt.Sprintf("%s tags=%v", fnType.String(), cfg.tags), nil
}

func describePopulate(n populateNode) (string, error) {
	if len(n.targets) == 0 {
		return "<empty>", nil
	}
	var cfg paramConfig
	if err := applyParamOptions(n.opts, &cfg); err != nil {
		return "", err
	}
	parts := make([]string, 0, len(n.targets))
	for _, target := range n.targets {
		typ := reflect.TypeOf(target)
		if typ == nil {
			parts = append(parts, "<nil>")
			continue
		}
		parts = append(parts, typ.String())
	}
	label := strings.Join(parts, ", ")
	if len(cfg.tags) == 0 {
		return label, nil
	}
	return fmt.Sprintf("%s tags=%v", label, cfg.tags), nil
}

func describeDecorate(n decorateNode) (string, error) {
	fnType := reflect.TypeOf(n.function)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return "", fmt.Errorf(errDecorateFunctionRequired)
	}
	explicit, hasExplicit, err := explicitTagSets(n, fnType)
	if err != nil {
		return "", err
	}
	if !hasExplicit {
		return fnType.String(), nil
	}
	return fmt.Sprintf("%s tags=%s", fnType.String(), formatTagSets(explicit)), nil
}

func formatTypeAndTags(typ reflect.Type, tags []tagSet) string {
	if typ == nil {
		return "<nil>"
	}
	if len(tags) == 0 {
		return typ.String()
	}
	return fmt.Sprintf("%s tags=%s", typ.String(), formatTagSets(tags))
}

func formatTagSets(tags []tagSet) string {
	parts := make([]string, 0, len(tags))
	for _, ts := range tags {
		label := ts.typString()
		if ts.name != "" {
			label += " name=" + ts.name
		}
		if ts.group != "" {
			label += " group=" + ts.group
		}
		parts = append(parts, label)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (ts tagSet) typString() string {
	if ts.typ == nil {
		return "<nil>"
	}
	return ts.typ.String()
}
