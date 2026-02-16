package us

import (
	"reflect"

	"github.com/bronystylecrazy/ultrastructure/caching/rd"
	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/imgutil"
	"github.com/bronystylecrazy/ultrastructure/lifecycle"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/bronystylecrazy/ultrastructure/storage/s3"
	"github.com/bronystylecrazy/ultrastructure/web"
	"go.uber.org/fx"
)

var nodeType = reflect.TypeOf((*di.Node)(nil)).Elem()

type App struct {
	nodes []any
}

func New(nodes ...any) *App {
	allNodes := append(defaultNodes(), flattenNodes(nodes)...)
	allNodes = append(allNodes, di.Invoke(cmd.RegisterCommands))
	return &App{nodes: allNodes}
}

func (a *App) Build() fx.Option {
	syncMeta()
	return di.App(a.nodes...).Build()
}

func (a *App) Run() error {
	syncMeta()
	return di.Run(a.nodes...)
}

func defaultNodes() []any {
	return []any{
		di.Diagnostics(),
		otel.Module(),
		lifecycle.Module(),
		web.Module(),
		database.Module(),
		realtime.Module(),
		cmd.Module(),
		s3.Module(),
		rd.Module(),
		imgutil.Module(),
	}
}

func flattenNodes(items []any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		appendFlattenedNode(&out, item)
	}
	return out
}

func appendFlattenedNode(out *[]any, item any) {
	switch v := item.(type) {
	case nil:
		return
	case []di.Node:
		for _, n := range v {
			appendFlattenedNode(out, n)
		}
		return
	case []any:
		for _, n := range v {
			appendFlattenedNode(out, n)
		}
		return
	}

	rv := reflect.ValueOf(item)
	if rv.IsValid() {
		k := rv.Kind()
		if (k == reflect.Slice || k == reflect.Array) && rv.Type().Elem().Implements(nodeType) {
			for i := range rv.Len() {
				appendFlattenedNode(out, rv.Index(i).Interface())
			}
			return
		}
	}

	*out = append(*out, item)
}
