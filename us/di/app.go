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
		case fx.Option:
			out = append(out, fxOptionNode{opt: v})
		default:
			out = append(out, errorNode{err: fmt.Errorf("unsupported node type %T", it)})
		}
	}
	return out
}

type fxOptionNode struct {
	opt fx.Option
}

func (n fxOptionNode) Build() (fx.Option, error) { return n.opt, nil }

type errorNode struct {
	err error
}

func (n errorNode) Build() (fx.Option, error) { return nil, n.err }
