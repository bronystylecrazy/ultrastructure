package us

import (
	"os"
	"reflect"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/imgutil"
	"github.com/bronystylecrazy/ultrastructure/lc"
	"github.com/bronystylecrazy/ultrastructure/meta"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/bronystylecrazy/ultrastructure/security/token"
	"github.com/bronystylecrazy/ultrastructure/storage/s3"
	"github.com/bronystylecrazy/ultrastructure/web"
	kservice "github.com/kardianos/service"
	"go.uber.org/fx"
)

var nodeType = reflect.TypeOf((*di.Node)(nil)).Elem()

type App struct {
	nodes               []any
	enableServiceHost   bool
	serviceCommandToken string
}

func New(nodes ...any) *App {
	app := &App{
		serviceCommandToken: "service",
	}
	var extras []any
	for _, node := range nodes {
		if opt, ok := node.(appOption); ok {
			opt.apply(app)
			continue
		}
		extras = append(extras, node)
	}

	allNodes := append(defaultNodes(), flattenNodes(extras)...)
	allNodes = append(allNodes, di.Invoke(cmd.RegisterCommands))
	app.nodes = allNodes
	return app
}

func (a *App) Build() fx.Option {
	syncMeta()
	return di.App(a.nodes...).Build()
}

func (a *App) Run() error {
	syncMeta()
	if a.enableServiceHost && shouldRunServiceHost(os.Args[1:], a.serviceCommandToken) {
		return a.runWithServiceHost()
	}
	return di.Run(a.nodes...)
}

func (a *App) runWithServiceHost() error {
	program := &serviceHostProgram{owner: a}
	svc, err := kservice.New(program, &kservice.Config{
		Name:        sanitizeServiceName(meta.Name),
		DisplayName: meta.Name,
		Description: meta.Description,
	})
	if err != nil {
		return err
	}
	return svc.Run()
}

func shouldRunServiceHost(args []string, serviceCommandToken string) bool {
	cmdToken, ok := firstCommandToken(args)
	if !ok {
		return true
	}
	if cmdToken == "serve" {
		return true
	}
	if cmdToken == strings.TrimSpace(serviceCommandToken) {
		return false
	}
	return false
}

func firstCommandToken(args []string) (string, bool) {
	for _, arg := range args {
		v := strings.TrimSpace(arg)
		if v == "" || strings.HasPrefix(v, "-") {
			continue
		}
		return v, true
	}
	return "", false
}

func sanitizeServiceName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		return "ultrastructure"
	}
	return strings.ReplaceAll(base, " ", "-")
}

func defaultNodes() []any {
	return []any{
		di.Diagnostics(),
		otel.Module(),
		lc.Module(),
		web.Provide(),
		token.Module(),
		database.Module(),
		realtime.Module(),
		cmd.Module(),
		s3.Module(),
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
