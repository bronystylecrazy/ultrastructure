package di

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
)

// Node is a declarative DI node that can be built into fx.Options.
type Node interface {
	Build() (fx.Option, error)
}

type appNode struct {
	nodes []Node
}

type restartSignal chan struct{}

// App collects declarative nodes and builds them into an fx.Option.
func App(nodes ...any) *appNode {
	return &appNode{nodes: collectNodes(nodes)}
}

// Run builds and runs the app, restarting on config changes.
func Run(nodes ...any) error {
	return App(nodes...).Run()
}

func (a *appNode) Build() fx.Option {
	nextID := 0
	nextScopeID := 1
	nodes := applyAutoGroups(a.nodes, nil)
	nodes = applyConfigScopes(nodes)
	nodes = applyAutoInjectFields(nodes, false)
	if cfg, ok := collectGlobalConfigWatch(nodes); ok {
		nodes = applyGlobalConfigWatch(nodes, cfg)
	}
	resolver := buildConfigResolver(nodes)
	nodes = attachConfigResolvers(nodes, resolver)
	nodes, err := applyReplacements(nodes, nil, &nextID, &nextScopeID, []int{0}, nil)
	if err != nil {
		return fx.Error(err)
	}
	hasDiagnostics := false
	for _, n := range nodes {
		if _, ok := n.(diagnosticsNode); ok {
			hasDiagnostics = true
			break
		}
	}
	_, decorators, err := collectGlobalTagSets(nodes)
	if err != nil {
		return fx.Error(err)
	}
	var opts []fx.Option
	if hasDiagnostics {
		opts = append(opts, fx.RecoverFromPanics())
	}
	for _, n := range nodes {
		var opt fx.Option
		if _, ok := n.(decorateNode); ok {
			continue
		}
		var err error
		opt, err = n.Build()
		if err != nil {
			return fx.Error(err)
		}
		opts = append(opts, opt)
	}

	decorateOpts, err := buildDecorators(decorators)
	if err != nil {
		return fx.Error(err)
	}
	opts = append(opts, decorateOpts...)
	if len(opts) == 0 {
		return fx.Options()
	}
	if len(opts) == 1 {
		return opts[0]
	}
	return fx.Options(opts...)
}

// Run starts the app and restarts it when ConfigWatch triggers.
func (a *appNode) Run() error {
	restart := make(restartSignal, 1)
	for {
		app := fx.New(
			fx.Supply(restart),
			a.Build(),
		)
		if err := app.Start(context.Background()); err != nil {
			return err
		}
		select {
		case <-restart:
			stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			_ = app.Stop(stopCtx)
			cancel()
			continue
		case <-app.Done():
			stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			_ = app.Stop(stopCtx)
			cancel()
			return nil
		}
	}
}

func collectNodes(items []any) []Node {
	var out []Node
	for _, it := range items {
		switch v := it.(type) {
		case nil:
			continue
		case Node:
			out = append(out, v)
		case configOption:
			out = append(out, configOptionNode{opt: v})
		case ConfigWatchOption:
			out = append(out, configOptionNode{watch: v})
		case fx.Option:
			out = append(out, fxOptionNode{opt: v})
		default:
			out = append(out, errorNode{err: fmt.Errorf("unsupported node type %T", it)})
		}
	}
	return out
}

func applyConfigScopes(nodes []Node) []Node {
	last := findLastConfigFileInScope(nodes)
	nextID := 0
	if last >= 0 {
		scope := nextConfigScope(&nextID)
		idx := 0
		return applyScopedConfigFilesInScope(nodes, scope, scope, &idx, last, &nextID, false)
	}
	return applyScopedConfigFiles(nodes, "", &nextID)
}

func nextConfigScope(nextID *int) string {
	id := *nextID
	*nextID++
	return fmt.Sprintf("cfgscope-%d", id)
}

func findLastConfigFileInScope(nodes []Node) int {
	idx := 0
	last := -1
	findLastConfigFileInScopeAt(nodes, &idx, &last)
	return last
}

func findLastConfigFileInScopeAt(nodes []Node, idx *int, last *int) {
	for _, n := range nodes {
		switch v := n.(type) {
		case configFileNode:
			*last = *idx
			*idx++
		case moduleNode:
			// module is a new scope; ignore during scan
			continue
		case optionsNode:
			findLastConfigFileInScopeAt(v.nodes, idx, last)
		case conditionalNode:
			findLastConfigFileInScopeAt(v.nodes, idx, last)
		case switchNode:
			for _, c := range v.cases {
				findLastConfigFileInScopeAt(c.nodes, idx, last)
			}
			findLastConfigFileInScopeAt(v.defaultCase.nodes, idx, last)
		}
	}
}

func applyScopedConfigFiles(nodes []Node, inheritedScope string, nextID *int) []Node {
	idx := 0
	last := findLastConfigFileInScope(nodes)
	scope := ""
	if last >= 0 {
		scope = nextConfigScope(nextID)
	}
	effective := scope
	if effective == "" {
		effective = inheritedScope
	}
	return applyScopedConfigFilesInScope(nodes, effective, scope, &idx, last, nextID, true)
}

func applyScopedConfigFilesInScope(nodes []Node, effectiveScope string, localScope string, idx *int, last int, nextID *int, allowModuleConfig bool) []Node {
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		switch v := n.(type) {
		case configFileNode:
			if *idx == last {
				out = append(out, withConfigScope(v, localScope))
			}
			*idx++
		case moduleNode:
			if allowModuleConfig {
				v.nodes = applyScopedConfigFiles(v.nodes, effectiveScope, nextID)
			} else {
				v.nodes = applyScopeToNodes(v.nodes, effectiveScope)
			}
			out = append(out, v)
		case optionsNode:
			v.nodes = applyScopedConfigFilesInScope(v.nodes, effectiveScope, localScope, idx, last, nextID, allowModuleConfig)
			out = append(out, v)
		case conditionalNode:
			v.nodes = applyScopedConfigFilesInScope(v.nodes, effectiveScope, localScope, idx, last, nextID, allowModuleConfig)
			out = append(out, v)
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for i, c := range v.cases {
				c.nodes = applyScopedConfigFilesInScope(c.nodes, effectiveScope, localScope, idx, last, nextID, allowModuleConfig)
				cases[i] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: applyScopedConfigFilesInScope(v.defaultCase.nodes, effectiveScope, localScope, idx, last, nextID, allowModuleConfig)}
			out = append(out, v)
		default:
			out = append(out, withConfigScope(n, effectiveScope))
		}
	}
	return out
}

func applyScopeToNodes(nodes []Node, scope string) []Node {
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		switch v := n.(type) {
		case configFileNode:
			continue
		case moduleNode:
			v.nodes = applyScopeToNodes(v.nodes, scope)
			out = append(out, v)
		case optionsNode:
			v.nodes = applyScopeToNodes(v.nodes, scope)
			out = append(out, v)
		case conditionalNode:
			v.nodes = applyScopeToNodes(v.nodes, scope)
			out = append(out, v)
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for i, c := range v.cases {
				c.nodes = applyScopeToNodes(c.nodes, scope)
				cases[i] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: applyScopeToNodes(v.defaultCase.nodes, scope)}
			out = append(out, v)
		default:
			out = append(out, withConfigScope(n, scope))
		}
	}
	return out
}

func withConfigScope(n Node, scope string) Node {
	if scoper, ok := n.(configScoper); ok {
		return scoper.withConfigScope(scope)
	}
	return n
}

type fxOptionNode struct {
	opt fx.Option
}

func (n fxOptionNode) Build() (fx.Option, error) { return n.opt, nil }

type errorNode struct {
	err error
}

func (n errorNode) Build() (fx.Option, error) { return nil, n.err }
