package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

type fxDecoDep struct {
	val string
}

func TestDecorateWithFxInDeps(t *testing.T) {
	var got *basicThing
	app := fx.New(
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got == nil || got.value != "provided-dep" {
		t.Fatalf("unexpected decorated value: %#v", got)
	}
}

func TestDecorateMixedFxInAndParams(t *testing.T) {
	app := fx.New(
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
	}
}
