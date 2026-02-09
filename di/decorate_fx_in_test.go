package di

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type fxDecoDep struct {
	val string
}

func TestDecorateWithFxInDeps(t *testing.T) {
	var got *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing),
			Supply(&fxDecoDep{val: "dep"}, Name("dep")),
			Decorate(func(b *basicThing, in struct {
				fx.In
				Dep *fxDecoDep `name:"dep"`
			}) *basicThing {
				b.value = b.value + "-" + in.Dep.val
				return b
			}),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided-dep" {
		t.Fatalf("unexpected decorated value: %#v", got)
	}
}

func TestDecorateMixedFxInAndParams(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Provide(newBasicThing),
			Supply(&fxDecoDep{val: "-fxin"}, Name("dep")),
			Decorate(func(in struct {
				fx.In
				Target *basicThing
				Dep    *fxDecoDep `name:"dep"`
			}) *basicThing {
				in.Target.value = in.Target.value + in.Dep.val
				return in.Target
			}),
			Decorate(func(b *basicThing) *basicThing {
				b.value = b.value + "-plain"
				return b
			}),
		).Build(),
	)
	if err := app.Start(context.Background()); err == nil {
		_ = app.Stop(context.Background())
		t.Fatal("expected start to fail for mixed fx.In and positional decorators")
	} else if !strings.Contains(err.Error(), errDecorateSignatureMismatch) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsFxInAsOnlyParam(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Provide(newBasicThing),
			Decorate(func(in struct {
				fx.In
				Target *basicThing
			}) *basicThing {
				in.Target.value = in.Target.value + "-fxin"
				return in.Target
			}),
		).Build(),
	)
	if err := app.Start(context.Background()); err == nil {
		_ = app.Stop(context.Background())
		t.Fatal("expected start to fail for fx.In-only decorator")
	} else if !strings.Contains(err.Error(), errDecorateSignatureMismatch) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsFxInWithParamTags(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Provide(newBasicThing),
			Supply(&fxDecoDep{val: "-fxin"}, Name("dep")),
			Decorate(func(b *basicThing, in struct {
				fx.In
				Dep *fxDecoDep `name:"dep"`
			}) *basicThing {
				b.value = b.value + in.Dep.val
				return b
			}, Params(`name:"dep"`)),
		).Build(),
	)
	if err := app.Start(context.Background()); err == nil {
		_ = app.Stop(context.Background())
		t.Fatal("expected start to fail for fx.In with ParamTags")
	} else if !strings.Contains(err.Error(), errDecorateSignatureMismatch) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsMultipleFxInDecorators(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Provide(newBasicThing),
			Supply(&fxDecoDep{val: "-one"}, Name("dep1")),
			Supply(&fxDecoDep{val: "-two"}, Name("dep2")),
			Decorate(func(b *basicThing, in struct {
				fx.In
				Dep *fxDecoDep `name:"dep1"`
			}) *basicThing {
				b.value = b.value + in.Dep.val
				return b
			}),
			Decorate(func(b *basicThing, in struct {
				fx.In
				Dep *fxDecoDep `name:"dep2"`
			}) *basicThing {
				b.value = b.value + in.Dep.val
				return b
			}),
		).Build(),
	)
	if err := app.Start(context.Background()); err == nil {
		_ = app.Stop(context.Background())
		t.Fatal("expected start to fail for multiple fx.In decorators")
	} else if !strings.Contains(err.Error(), errDecorateSignatureMismatch) {
		t.Fatalf("unexpected error: %v", err)
	}
}
