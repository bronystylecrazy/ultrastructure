package di

import (
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type fxInDep struct {
	val string
}

type fxInTarget struct {
	Dep *fxInDep
}

type fxOutThing struct {
	val string
}

type fxOutResult struct {
	fx.Out
	Thing *fxOutThing `name:"thing"`
}

func TestFxInStructInjection(t *testing.T) {
	var got *fxInTarget
	app := fxtest.New(t,
		App(
			Supply(&fxInDep{val: "dep"}),
			Provide(func(in struct {
				fx.In
				Dep *fxInDep
			}) *fxInTarget {
				return &fxInTarget{Dep: in.Dep}
			}),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.Dep == nil || got.Dep.val != "dep" {
		t.Fatalf("unexpected target: %#v", got)
	}
}

func TestFxOutProvidesNamedValue(t *testing.T) {
	var got *fxOutThing
	app := fxtest.New(t,
		App(
			Provide(func() fxOutResult {
				return fxOutResult{Thing: &fxOutThing{val: "ok"}}
			}),
			Populate(&got, Name("thing")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.val != "ok" {
		t.Fatalf("unexpected thing: %#v", got)
	}
}

type fxAutoHandler interface {
	Name() string
}

type fxAutoHandlerImpl struct {
	name string
}

func (h *fxAutoHandlerImpl) Name() string { return h.name }

func TestFxInWithAutoGroup(t *testing.T) {
	var count int
	app := fxtest.New(t,
		App(
			AutoGroup[fxAutoHandler]("handlers"),
			Supply(&fxAutoHandlerImpl{name: "h1"}),
			Invoke(func(in struct {
				fx.In
				Handlers []fxAutoHandler `group:"handlers"`
			}) {
				count = len(in.Handlers)
			}),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if count != 1 {
		t.Fatalf("expected 1 handler, got %d", count)
	}
}

type fxOutHandlerResult struct {
	fx.Out
	Handler fxAutoHandler `group:"handlers"`
}

func TestFxOutWithAutoGroup(t *testing.T) {
	var count int
	app := fxtest.New(t,
		App(
			AutoGroup[fxAutoHandler]("handlers"),
			Provide(func() fxOutHandlerResult {
				return fxOutHandlerResult{Handler: &fxAutoHandlerImpl{name: "h1"}}
			}),
			Invoke(func(in struct {
				fx.In
				Handlers []fxAutoHandler `group:"handlers"`
			}) {
				count = len(in.Handlers)
			}),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if count != 1 {
		t.Fatalf("expected 1 handler, got %d", count)
	}
}
