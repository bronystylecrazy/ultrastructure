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
	// Apply graph-wide transformations before building.
	nodes := applyAutoGroups(a.nodes, nil)
	nodes = applyConfigScopes(nodes)
	nodes = applyAutoInjectFields(nodes, false)
	// Expand global config watch settings into each config node.
	if cfg, ok := collectGlobalConfigWatch(nodes); ok {
		nodes = applyGlobalConfigWatch(nodes, cfg)
	}
	// Attach config resolvers so Switch/If can evaluate with config values.
	resolver := buildConfigResolver(nodes)
	nodes = attachConfigResolvers(nodes, resolver)
	// Apply replacements after scopes and resolvers are in place.
	nodes, err := applyReplacements(nodes, nil, &nextID, &nextScopeID, []int{0}, nil)
	if err != nil {
		return fx.Error(err)
	}
	// Detect diagnostics nodes to enable panic recovery.
	hasDiagnostics := false
	for _, n := range nodes {
		if _, ok := n.(diagnosticsNode); ok {
			hasDiagnostics = true
			break
		}
	}
	// Extract diagnostics options so they can be appended on error paths.
	diagOpts, nodes := extractDiagnostics(nodes)
	// Collect tag sets and decorate entries after all transformations.
	_, decorators, err := collectGlobalTagSets(nodes)
	if err != nil {
		return fx.Options(append(diagOpts, fx.Error(err))...)
	}
	var opts []fx.Option
	if hasDiagnostics {
		// Match fx.App defaults when diagnostics is enabled.
		opts = append(opts, fx.RecoverFromPanics())
	}
	for _, n := range nodes {
		var opt fx.Option
		if _, ok := n.(decorateNode); ok {
			// Decorators are composed globally below.
			continue
		}
		var err error
		opt, err = n.Build()
		if err != nil {
			return fx.Options(append(append(opts, diagOpts...), fx.Error(err))...)
		}
		opts = append(opts, opt)
	}

	decorateOpts, err := buildDecorators(decorators)
	if err != nil {
		return fx.Options(append(append(opts, diagOpts...), fx.Error(err))...)
	}
	// Append composed decorators last.
	opts = append(opts, decorateOpts...)
	// Append diagnostics options last so they win logger selection.
	opts = append(opts, diagOpts...)
	if len(opts) == 0 {
		return fx.Options()
	}
	if len(opts) == 1 {
		return opts[0]
	}
	return fx.Options(opts...)
}

func extractDiagnostics(nodes []Node) ([]fx.Option, []Node) {
	if len(nodes) == 0 {
		return nil, nodes
	}
	var diagOpts []fx.Option
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		if _, ok := n.(diagnosticsNode); ok {
			// Build diagnostics immediately to include in error paths.
			opt, err := n.Build()
			if err != nil {
				diagOpts = append(diagOpts, fx.Error(err))
			} else {
				diagOpts = append(diagOpts, opt)
			}
			continue
		}
		out = append(out, n)
	}
	return diagOpts, out
}

// Run starts the app and restarts it when ConfigWatch triggers.
func (a *appNode) Run() error {
	restart := make(restartSignal, 1)
	for {
		// Rebuild the app each time a config watch triggers a restart.
		app := fx.New(
			fx.Supply(restart),
			a.Build(),
		)
		if err := app.Start(context.Background()); err != nil {
			return err
		}
		select {
		case <-restart:
			// Gracefully stop and rebuild on config changes.
			stopApp(app)
			continue
		case <-app.Done():
			// Stop when the app exits normally.
			stopApp(app)
			return nil
		}
	}
}

func stopApp(app *fx.App) {
	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	_ = app.Stop(stopCtx)
	cancel()
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
			// Allow config options at top-level for convenience.
			out = append(out, configOptionNode{opt: v})
		case ConfigWatchOption:
			// Allow watch options at top-level to configure global watchers.
			out = append(out, configOptionNode{watch: v})
		case fx.Option:
			// Accept raw fx.Options inside di.App.
			out = append(out, fxOptionNode{opt: v})
		default:
			out = append(out, errorNode{err: fmt.Errorf(errUnsupportedNodeType, it)})
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
