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

type Nodes []Node

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
	nodes = applyAutoInjectFields(nodes, false)
	// Apply replacements after scopes and resolvers are in place.
	nodes, err := applyReplacements(nodes, nil, &nextID, &nextScopeID, []int{0}, nil)
	if err != nil {
		return fx.Error(err)
	}
	priorityGroups, hasPriority := collectPriorityGroups(nodes)
	if hasPriority {
		orderCounter := 0
		nodes = applyAutoGroupOrderMetadata(nodes, &orderCounter, true)
		if len(priorityGroups) > 0 {
			nodes = appendAutoGroupOrderDecorators(nodes, priorityGroups)
		}
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
	// Collect tag sets and decorate entries only when decorators are present.
	var decorators []decorateEntry
	if hasDecorators(nodes) {
		_, decs, err := collectGlobalTagSets(nodes)
		if err != nil {
			return fx.Options(append(diagOpts, fx.Error(err))...)
		}
		decorators = decs
	}
	var opts []fx.Option
	var batchProvide []any
	var batchSupply []any
	batchPrivate := false
	batchKind := ""

	flushBatch := func() {
		if len(batchProvide) > 0 {
			if batchPrivate {
				opts = append(opts, fx.Provide(append(batchProvide, fx.Private)...))
			} else {
				opts = append(opts, fx.Provide(batchProvide...))
			}
		} else if len(batchSupply) > 0 {
			if batchPrivate {
				opts = append(opts, fx.Supply(append(batchSupply, fx.Private)...))
			} else {
				opts = append(opts, fx.Supply(batchSupply...))
			}
		}
		batchProvide = nil
		batchSupply = nil
		batchKind = ""
		batchPrivate = false
	}

	addProvide := func(ctor any, private bool) {
		if batchKind != "provide" || batchPrivate != private {
			flushBatch()
			batchKind = "provide"
			batchPrivate = private
		}
		batchProvide = append(batchProvide, ctor)
	}

	addSupply := func(value any, private bool) {
		if batchKind != "supply" || batchPrivate != private {
			flushBatch()
			batchKind = "supply"
			batchPrivate = private
		}
		batchSupply = append(batchSupply, value)
	}

	if hasDiagnostics {
		// Match fx.App defaults when diagnostics is enabled.
		opts = append(opts, fx.RecoverFromPanics())
	}
	for _, n := range nodes {
		switch v := n.(type) {
		case decorateNode:
			// Decorators are composed globally below.
			continue
		case provideNode:
			ctor, private, extra, err := v.buildConstructor()
			if err != nil {
				flushBatch()
				return fx.Options(append(append(opts, diagOpts...), fx.Error(err))...)
			}
			addProvide(ctor, private)
			if len(extra) > 0 {
				flushBatch()
				opts = append(opts, extra...)
			}
			continue
		case supplyNode:
			ctor, value, useSupply, private, extra, err := v.buildSupply()
			if err != nil {
				flushBatch()
				return fx.Options(append(append(opts, diagOpts...), fx.Error(err))...)
			}
			if useSupply {
				addSupply(value, private)
			} else {
				addProvide(ctor, private)
			}
			if len(extra) > 0 {
				flushBatch()
				opts = append(opts, extra...)
			}
			continue
		default:
			flushBatch()
			opt, err := n.Build()
			if err != nil {
				return fx.Options(append(append(opts, diagOpts...), fx.Error(err))...)
			}
			opts = append(opts, opt)
		}
	}
	flushBatch()

	if len(decorators) > 0 {
		decorateOpts, err := buildDecorators(decorators)
		if err != nil {
			return fx.Options(append(append(opts, diagOpts...), fx.Error(err))...)
		}
		// Append composed decorators last.
		opts = append(opts, decorateOpts...)
	}
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
		case []Node:
			out = append(out, collectNodes(ConvertAnys(v))...)
		case []any:
			out = append(out, collectNodes(v)...)
		case Node:
			out = append(out, v)
		case fx.Option:
			// Accept raw fx.Options inside di.App.
			out = append(out, fxOptionNode{opt: v})
		default:
			out = append(out, errorNode{err: fmt.Errorf(errUnsupportedNodeType, it)})
		}
	}
	return out
}

func hasDecorators(nodes []Node) bool {
	for _, n := range nodes {
		switch v := n.(type) {
		case decorateNode:
			return true
		case provideNode:
			if optsHaveDecorate(v.opts) {
				return true
			}
		case supplyNode:
			if optsHaveDecorate(v.opts) {
				return true
			}
		case moduleNode:
			if hasDecorators(v.nodes) {
				return true
			}
		case optionsNode:
			if hasDecorators(v.nodes) {
				return true
			}
		case conditionalNode:
			if hasDecorators(v.nodes) {
				return true
			}
		case switchNode:
			for _, c := range v.cases {
				if hasDecorators(c.nodes) {
					return true
				}
			}
			if hasDecorators(v.defaultCase.nodes) {
				return true
			}
		}
	}
	return false
}

func optsHaveDecorate(opts []any) bool {
	for _, opt := range opts {
		if _, ok := opt.(decorateNode); ok {
			return true
		}
	}
	return false
}

type fxOptionNode struct {
	opt fx.Option
}

func (n fxOptionNode) Build() (fx.Option, error) { return n.opt, nil }

type errorNode struct {
	err error
}

func (n errorNode) Build() (fx.Option, error) { return nil, n.err }
