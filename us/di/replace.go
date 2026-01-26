package di

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/us"
	"go.uber.org/fx"
)

func applyReplacements(
	nodes []Node,
	inherited []replaceSpec,
	nextID *int,
	nextScopeID *int,
	scopeStack []int,
	provides []provideItem,
) ([]Node, error) {
	scopeID := scopeStack[len(scopeStack)-1]
	if provides == nil {
		var err error
		provides, err = collectScopeProvides(nodes)
		if err != nil {
			return nil, err
		}
	}

	localSpecs := []replaceSpec{}
	localByIndex := map[int]replaceSpec{}
	for i, n := range nodes {
		switch r := n.(type) {
		case replaceNode:
			spec, err := buildReplaceSpec(r, i)
			if err != nil {
				return nil, err
			}
			spec.depth = len(scopeStack) - 1
			spec.scopeID = scopeID
			spec.id = *nextID
			*nextID++
			localSpecs = append(localSpecs, spec)
			localByIndex[i] = spec
		case defaultNode:
			spec, err := buildDefaultSpec(r, i)
			if err != nil {
				return nil, err
			}
			spec.depth = len(scopeStack) - 1
			spec.scopeID = scopeID
			spec.id = *nextID
			*nextID++
			localSpecs = append(localSpecs, spec)
			localByIndex[i] = spec
		}
	}

	specs := append([]replaceSpec{}, inherited...)
	specs = append(specs, localSpecs...)

	final := make([]Node, 0, len(nodes))
	for i, n := range nodes {
		switch v := n.(type) {
		case moduleNode:
			childScopeID := *nextScopeID
			*nextScopeID++
			childStack := append(append([]int{}, scopeStack...), childScopeID)
			active := applicableSpecsAtIndex(specs, i)
			child, err := applyReplacements(v.nodes, inheritSpecs(active), nextID, nextScopeID, childStack, nil)
			if err != nil {
				return nil, err
			}
			final = append(final, moduleNode{name: v.name, nodes: child})
		case optionsNode:
			active := applicableSpecsAtIndex(specs, i)
			child, err := applyReplacements(v.nodes, inheritSpecs(active), nextID, nextScopeID, scopeStack, provides)
			if err != nil {
				return nil, err
			}
			final = append(final, optionsNode{nodes: child})
		case switchNode:
			selected, err := v.selectNodes()
			if err != nil {
				return nil, err
			}
			if len(selected) == 0 {
				final = append(final, switchNode{resolved: true})
				continue
			}
			active := applicableSpecsAtIndex(specs, i)
			child, err := applyReplacements(selected, inheritSpecs(active), nextID, nextScopeID, scopeStack, provides)
			if err != nil {
				return nil, err
			}
			final = append(final, switchNode{
				resolved: true,
				selected: child,
			})
		case conditionalNode:
			ok, err := v.eval()
			if err != nil {
				return nil, err
			}
			if !ok {
				final = append(final, v)
				continue
			}
			active := applicableSpecsAtIndex(specs, i)
			child, err := applyReplacements(v.nodes, inheritSpecs(active), nextID, nextScopeID, scopeStack, provides)
			if err != nil {
				return nil, err
			}
			final = append(final, conditionalNode{
				mode:      v.mode,
				cond:      v.cond,
				when:      v.when,
				nodes:     child,
				evaluated: true,
				result:    true,
			})
		case replaceNode:
			spec, ok := localByIndex[i]
			if !ok {
				continue
			}
			targets := replacementTargets(spec, provides)
			for _, ts := range targets {
				scoped := scopedTagSet(ts, spec.id)
				node, err := replacementNodeWithTagSet(spec, scoped)
				if err != nil {
					return nil, err
				}
				final = append(final, node)
			}
		case defaultNode:
			spec, ok := localByIndex[i]
			if !ok {
				continue
			}
			targets := defaultTargets(spec, provides, specs)
			for _, ts := range targets {
				scoped := scopedTagSet(ts, spec.id)
				node, err := replacementNodeWithTagSet(spec, scoped)
				if err != nil {
					return nil, err
				}
				final = append(final, node)
			}
		case provideNode:
			final = append(final, n)
		case supplyNode:
			final = append(final, n)
		case invokeNode:
			active := applicableSpecsAtIndex(specs, i)
			activeTags := buildActiveTagMap(provides, active)
			node, changed, err := rewriteInvokeWithTags(v, activeTags)
			if err != nil {
				return nil, err
			}
			if changed {
				final = append(final, node)
			} else {
				final = append(final, n)
			}
		default:
			final = append(final, n)
		}
	}

	return final, nil
}

// Replace declares a replacement value for the entire scope.
func Replace(value any, opts ...any) Node {
	return replaceNode{value: value, opts: opts, mode: replaceAll}
}

// ReplaceBefore applies only to nodes declared before it in the same scope.
func ReplaceBefore(value any, opts ...any) Node {
	return replaceNode{value: value, opts: opts, mode: replaceBefore}
}

// ReplaceAfter applies only to nodes declared after it in the same scope.
func ReplaceAfter(value any, opts ...any) Node {
	return replaceNode{value: value, opts: opts, mode: replaceAfter}
}

type replaceNode struct {
	value any
	opts  []any
	mode  replaceMode
}

func (n replaceNode) Build() (fx.Option, error) {
	cfg, decorators, _, err := parseBindOptions(n.opts)
	if err != nil {
		return nil, err
	}
	if len(decorators) > 0 {
		return nil, fmt.Errorf("replace does not support decorate options")
	}
	if cfg.privateSet {
		return nil, fmt.Errorf("replace does not support private/public")
	}
	if cfg.pendingName != "" || cfg.pendingGroup != "" {
		return nil, fmt.Errorf("replace does not support named or grouped exports")
	}
	var provideOpts []us.ProvideOption
	if cfg.includeSelf {
		provideOpts = append(provideOpts, us.AsSelf())
	}
	for _, exp := range cfg.exports {
		if exp.grouped {
			return nil, fmt.Errorf("replace does not support groups")
		}
		if exp.named {
			return nil, fmt.Errorf("replace does not support named exports")
		}
		provideOpts = append(provideOpts, us.AsTypeOf(exp.typ))
	}
	if len(provideOpts) == 0 {
		return us.Replace(n.value), nil
	}
	args := make([]any, 0, 1+len(provideOpts))
	args = append(args, n.value)
	for _, opt := range provideOpts {
		args = append(args, opt)
	}
	return us.Replace(args...), nil
}

func buildReplaceSpec(n replaceNode, pos int) (replaceSpec, error) {
	cfg, _, _, err := parseBindOptions(n.opts)
	if err != nil {
		return replaceSpec{}, err
	}
	if cfg.privateSet {
		return replaceSpec{}, fmt.Errorf("replace does not support private/public")
	}
	var (
		tagSets []tagSet
		node    Node
	)
	if isFunc(n.value) {
		_, tagSets, err = buildProvideOptions(cfg, n.value, nil)
		if err != nil {
			return replaceSpec{}, err
		}
		node = provideNode{constructor: n.value, opts: n.opts}
	} else {
		_, tagSets, err = buildProvideOptions(cfg, nil, n.value)
		if err != nil {
			return replaceSpec{}, err
		}
		node = supplyNode{value: n.value, opts: n.opts}
	}
	if len(tagSets) == 0 {
		return replaceSpec{}, fmt.Errorf("replace requires a type match")
	}
	return replaceSpec{tagSets: tagSets, node: node, pos: pos, mode: n.mode}, nil
}

func buildDefaultSpec(n defaultNode, pos int) (replaceSpec, error) {
	cfg, _, _, err := parseBindOptions(n.opts)
	if err != nil {
		return replaceSpec{}, err
	}
	if cfg.privateSet {
		return replaceSpec{}, fmt.Errorf("default does not support private/public")
	}
	var (
		tagSets []tagSet
		node    Node
	)
	if isFunc(n.value) {
		_, tagSets, err = buildProvideOptions(cfg, n.value, nil)
		if err != nil {
			return replaceSpec{}, err
		}
		node = provideNode{constructor: n.value, opts: n.opts}
	} else {
		_, tagSets, err = buildProvideOptions(cfg, nil, n.value)
		if err != nil {
			return replaceSpec{}, err
		}
		node = supplyNode{value: n.value, opts: n.opts}
	}
	if len(tagSets) == 0 {
		return replaceSpec{}, fmt.Errorf("default requires a type match")
	}
	return replaceSpec{tagSets: tagSets, node: node, pos: pos, mode: replaceAll, isDefault: true}, nil
}

func matchesReplace(tagSets []tagSet, specs []replaceSpec) bool {
	_, ok := selectReplacement(tagSets, specs)
	return ok
}

func collectScopeProvides(nodes []Node) ([]provideItem, error) {
	var out []provideItem
	for _, n := range nodes {
		switch v := n.(type) {
		case provideNode:
			cfg, _, _, err := parseBindOptions(v.opts)
			if err != nil {
				return nil, err
			}
			_, tagSets, err := buildProvideOptions(cfg, v.constructor, nil)
			if err != nil {
				return nil, err
			}
			out = append(out, provideItem{node: v, tagSets: tagSets})
		case supplyNode:
			cfg, _, _, err := parseBindOptions(v.opts)
			if err != nil {
				return nil, err
			}
			_, tagSets, err := buildProvideOptions(cfg, nil, v.value)
			if err != nil {
				return nil, err
			}
			out = append(out, provideItem{node: v, tagSets: tagSets})
		case interface{ provideTagSets() ([]tagSet, error) }:
			tagSets, err := v.provideTagSets()
			if err != nil {
				return nil, err
			}
			out = append(out, provideItem{node: n, tagSets: tagSets})
		case optionsNode:
			child, err := collectScopeProvides(v.nodes)
			if err != nil {
				return nil, err
			}
			out = append(out, child...)
		case moduleNode:
			child, err := collectScopeProvides(v.nodes)
			if err != nil {
				return nil, err
			}
			out = append(out, child...)
		}
	}
	return out, nil
}

func matchingTagSets(provideTags []tagSet, replaceTags []tagSet) []tagSet {
	var out []tagSet
	for _, ts := range provideTags {
		for _, rt := range replaceTags {
			if !typesMatch(ts.typ, rt.typ) {
				continue
			}
			if rt.name != "" && ts.name != rt.name {
				continue
			}
			if rt.group != "" && ts.group != rt.group {
				continue
			}
			out = append(out, ts)
			break
		}
	}
	return out
}

func applicableSpecsAtIndex(specs []replaceSpec, index int) []replaceSpec {
	if len(specs) == 0 {
		return nil
	}
	out := make([]replaceSpec, 0, len(specs))
	for _, spec := range specs {
		if spec.isDefault {
			continue
		}
		if spec.inherited {
			out = append(out, spec)
			continue
		}
		switch spec.mode {
		case replaceAll:
			out = append(out, spec)
		case replaceAfter:
			if spec.pos < index {
				out = append(out, spec)
			}
		case replaceBefore:
			if spec.pos > index {
				out = append(out, spec)
			}
		}
	}
	return out
}

func inheritSpecs(specs []replaceSpec) []replaceSpec {
	if len(specs) == 0 {
		return nil
	}
	out := make([]replaceSpec, len(specs))
	for i, spec := range specs {
		spec.inherited = true
		spec.mode = replaceAll
		out[i] = spec
	}
	return out
}

func buildActiveTagMap(provides []provideItem, specs []replaceSpec) map[string]tagSet {
	if len(provides) == 0 || len(specs) == 0 {
		return map[string]tagSet{}
	}
	active := map[string]tagSet{}
	for _, p := range provides {
		for _, ts := range p.tagSets {
			spec, ok := selectReplacement([]tagSet{ts}, specs)
			if !ok {
				continue
			}
			active[fullTagKey(ts)] = scopedTagSet(ts, spec.id)
		}
	}
	return active
}

func defaultTargets(spec replaceSpec, provides []provideItem, specs []replaceSpec) []tagSet {
	seen := map[string]bool{}
	var out []tagSet
	add := func(ts tagSet) {
		key := fullTagKey(ts)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, ts)
	}
	for _, ts := range spec.tagSets {
		add(ts)
	}
	for _, p := range provides {
		for _, ts := range p.tagSets {
			if !matchesAny(tagSet{typ: ts.typ, name: ts.name, group: ts.group}, spec.tagSets) {
				continue
			}
			if matchesReplace([]tagSet{ts}, specs) {
				continue
			}
			add(ts)
		}
	}
	return out
}

func matchesAny(ts tagSet, specTags []tagSet) bool {
	for _, rt := range specTags {
		if !typesMatch(ts.typ, rt.typ) {
			continue
		}
		if rt.name != "" && ts.name != rt.name {
			continue
		}
		if rt.group != "" && ts.group != rt.group {
			continue
		}
		return true
	}
	return false
}

func replacementTargets(spec replaceSpec, provides []provideItem) []tagSet {
	seen := map[string]bool{}
	var out []tagSet
	add := func(ts tagSet) {
		key := fullTagKey(ts)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, ts)
	}
	for _, ts := range spec.tagSets {
		add(ts)
	}
	for _, p := range provides {
		matched := matchingTagSets(p.tagSets, spec.tagSets)
		for _, ts := range matched {
			add(ts)
		}
	}
	return out
}

func fullTagKey(ts tagSet) string {
	typeKey := ""
	if ts.typ != nil {
		typeKey = ts.typ.String()
	}
	return typeKey + "|n:" + ts.name + "|g:" + ts.group
}

func scopedTagSet(ts tagSet, replaceID int) tagSet {
	suffix := "__di_replace_" + strconv.Itoa(replaceID)
	if ts.name != "" {
		return tagSet{name: ts.name + suffix, typ: ts.typ}
	}
	if ts.group != "" {
		return tagSet{group: ts.group + suffix, typ: ts.typ}
	}
	return tagSet{name: scopedTypeName(ts.typ, replaceID), typ: ts.typ}
}

func scopedTypeName(typ reflect.Type, replaceID int) string {
	base := "unknown"
	if typ != nil {
		base = typ.String()
	}
	base = strings.NewReplacer("/", "_", ".", "_", "*", "ptr", "[", "_", "]", "_").Replace(base)
	return "__di_replace_" + strconv.Itoa(replaceID) + "__" + base
}

func selectReplacement(tagSets []tagSet, specs []replaceSpec) (replaceSpec, bool) {
	bestScore := -1
	bestDepth := 0
	bestIndex := -1
	var best replaceSpec
	for i, spec := range specs {
		score := bestReplaceScore(tagSets, spec.tagSets)
		if score < 0 {
			continue
		}
		if bestScore == -1 ||
			score > bestScore ||
			(score == bestScore && spec.depth < bestDepth) ||
			(score == bestScore && spec.depth == bestDepth && i > bestIndex) {
			bestScore = score
			bestDepth = spec.depth
			bestIndex = i
			best = spec
		}
	}
	if bestScore < 0 {
		return replaceSpec{}, false
	}
	return best, true
}

func bestReplaceScore(tagSets []tagSet, replaceTags []tagSet) int {
	best := -1
	for _, ts := range tagSets {
		for _, rt := range replaceTags {
			if !typesMatch(ts.typ, rt.typ) {
				continue
			}
			if rt.name != "" && ts.name != rt.name {
				continue
			}
			if rt.group != "" && ts.group != rt.group {
				continue
			}
			score := replaceSpecificity(rt)
			if score > best {
				best = score
			}
		}
	}
	return best
}

func replaceSpecificity(ts tagSet) int {
	if ts.name != "" && ts.group != "" {
		return 3
	}
	if ts.group != "" {
		return 2
	}
	if ts.name != "" {
		return 1
	}
	return 0
}

func replacementNodeWithTags(spec replaceSpec, provideTags []tagSet) (Node, []tagSet, error) {
	baseType, err := replacementBaseType(spec.node)
	if err != nil {
		return nil, nil, err
	}

	replaceHasName, replaceHasGroup := false, false
	for _, ts := range spec.tagSets {
		if !typesMatch(ts.typ, baseType) {
			continue
		}
		if ts.name != "" {
			replaceHasName = true
		}
		if ts.group != "" {
			replaceHasGroup = true
		}
	}

	provideNames := map[string]struct{}{}
	provideGroups := map[string]struct{}{}
	for _, ts := range provideTags {
		if !typesMatch(ts.typ, baseType) {
			continue
		}
		if ts.name != "" {
			provideNames[ts.name] = struct{}{}
		}
		if ts.group != "" {
			provideGroups[ts.group] = struct{}{}
		}
	}

	var extra []any
	if !replaceHasName {
		if name, ok := singleKey(provideNames); ok {
			extra = append(extra, Name(name))
		}
	}
	if !replaceHasGroup {
		if group, ok := singleKey(provideGroups); ok {
			extra = append(extra, Group(group))
		}
	}
	if len(extra) == 0 {
		return spec.node, spec.tagSets, nil
	}

	switch n := spec.node.(type) {
	case provideNode:
		opts := append([]any{}, n.opts...)
		opts = append(opts, extra...)
		node := provideNode{constructor: n.constructor, opts: opts}
		cfg, _, _, err := parseBindOptions(node.opts)
		if err != nil {
			return nil, nil, err
		}
		_, tagSets, err := buildProvideOptions(cfg, n.constructor, nil)
		if err != nil {
			return nil, nil, err
		}
		return node, tagSets, nil
	case supplyNode:
		opts := append([]any{}, n.opts...)
		opts = append(opts, extra...)
		node := supplyNode{value: n.value, opts: opts}
		cfg, _, _, err := parseBindOptions(node.opts)
		if err != nil {
			return nil, nil, err
		}
		_, tagSets, err := buildProvideOptions(cfg, nil, n.value)
		if err != nil {
			return nil, nil, err
		}
		return node, tagSets, nil
	default:
		return spec.node, spec.tagSets, nil
	}
}

func replacementNodeWithTagSet(spec replaceSpec, target tagSet) (Node, error) {
	if target.name == "" && target.group == "" {
		return spec.node, nil
	}
	switch n := spec.node.(type) {
	case provideNode:
		opts := overrideNameGroupOpts(n.opts, target)
		return provideNode{constructor: n.constructor, opts: opts}, nil
	case supplyNode:
		opts := overrideNameGroupOpts(n.opts, target)
		return supplyNode{value: n.value, opts: opts}, nil
	default:
		return spec.node, nil
	}
}

func overrideNameGroupOpts(opts []any, target tagSet) []any {
	filtered := make([]any, 0, len(opts)+1)
	for _, opt := range opts {
		switch opt.(type) {
		case nameOption, groupOption:
			continue
		default:
			filtered = append(filtered, opt)
		}
	}
	if target.name != "" {
		filtered = append(filtered, Name(target.name))
	}
	if target.group != "" {
		filtered = append(filtered, Group(target.group))
	}
	return filtered
}

func rewriteInvokeWithTags(node invokeNode, activeTags map[string]tagSet) (invokeNode, bool, error) {
	if len(activeTags) == 0 {
		return node, false, nil
	}
	fnType := reflect.TypeOf(node.function)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return node, false, nil
	}
	numIn := fnType.NumIn()
	if numIn == 0 {
		return node, false, nil
	}
	var cfg paramConfig
	for _, opt := range node.opts {
		if opt != nil {
			opt.applyParam(&cfg)
		}
		if cfg.err != nil {
			return node, false, cfg.err
		}
	}
	tags := make([]string, numIn)
	for i := 0; i < numIn && i < len(cfg.tags); i++ {
		tags[i] = cfg.tags[i]
	}
	changed := false
	for i := 0; i < numIn; i++ {
		paramType := fnType.In(i)
		tag := tags[i]
		name, group := parseTagNameGroup(tag)
		key := fullTagKey(tagSet{typ: paramType, name: name, group: group})
		scoped, ok := activeTags[key]
		if !ok {
			continue
		}
		newTag := rewriteParamTag(tag, scoped)
		if newTag != tag {
			tags[i] = newTag
			changed = true
		}
	}
	if !changed {
		return node, false, nil
	}
	return invokeNode{
		function:          node.function,
		opts:              node.opts,
		paramTagsOverride: tags,
	}, true, nil
}

func parseTagNameGroup(tag string) (string, string) {
	name, _ := extractTagValue(tag, `name:"`)
	group, _ := extractTagValue(tag, `group:"`)
	return name, group
}

func extractTagValue(tag string, key string) (string, bool) {
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

func rewriteParamTag(tag string, scoped tagSet) string {
	repl := ""
	if scoped.group != "" {
		repl = `group:"` + scoped.group + `"`
	} else if scoped.name != "" {
		repl = `name:"` + scoped.name + `"`
	} else {
		return tag
	}
	if strings.Contains(tag, `name:"`) {
		if updated, ok := replaceTagValue(tag, `name:"`, scoped.name); ok {
			return updated
		}
	}
	if strings.Contains(tag, `group:"`) {
		if updated, ok := replaceTagValue(tag, `group:"`, scoped.group); ok {
			return updated
		}
	}
	if tag == "" {
		return repl
	}
	return tag + " " + repl
}

func replaceTagValue(tag string, key string, value string) (string, bool) {
	idx := strings.Index(tag, key)
	if idx < 0 {
		return tag, false
	}
	start := idx + len(key)
	end := strings.Index(tag[start:], `"`)
	if end < 0 {
		return tag, false
	}
	end += start
	return tag[:start] + value + tag[end:], true
}

func replacementBaseType(node Node) (reflect.Type, error) {
	switch n := node.(type) {
	case provideNode:
		return constructorResultType(n.constructor)
	case supplyNode:
		if n.value == nil {
			return nil, fmt.Errorf("value must not be nil")
		}
		return reflect.TypeOf(n.value), nil
	default:
		return nil, fmt.Errorf("replacement must be provide or supply")
	}
}

func singleKey(values map[string]struct{}) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	for v := range values {
		return v, true
	}
	return "", false
}

func typesMatch(a, b reflect.Type) bool {
	if a == nil || b == nil {
		return false
	}
	return a == b
}

func isFunc(v any) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Func
}

type replaceMode int

const (
	replaceAll replaceMode = iota
	replaceBefore
	replaceAfter
)

type replaceSpec struct {
	tagSets   []tagSet
	node      Node
	pos       int
	id        int
	depth     int
	scopeID   int
	mode      replaceMode
	inherited bool
	isDefault bool
}
