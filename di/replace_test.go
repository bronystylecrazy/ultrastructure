package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

func TestReplaceOverridesProvide(t *testing.T) {
	var got *basicThing
	app := fx.New(
		App(
			Provide(newBasicThing),
			Replace(&basicThing{value: "replaced"}),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got == nil || got.value != "replaced" {
		t.Fatalf("unexpected replaced value: %#v", got)
	}
}

func TestReplaceBeforeAfterOrder(t *testing.T) {
	type firstThing struct {
		value string
	}
	type secondThing struct {
		value string
	}
	var first *firstThing
	var second *secondThing
	app := fx.New(
		App(
			Provide(func() *firstThing { return &firstThing{value: "orig"} }),
			Populate(&first),
			ReplaceBefore(&firstThing{value: "before"}),
			ReplaceAfter(&secondThing{value: "after"}),
			Provide(func() *secondThing { return &secondThing{value: "later"} }),
			Populate(&second),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if first == nil || first.value != "before" {
		t.Fatalf("unexpected first: %#v", first)
	}
	if second == nil || second.value != "after" {
		t.Fatalf("unexpected second: %#v", second)
	}
}

func TestReplaceBeforeModuleScope(t *testing.T) {
	var outside *basicThing
	var inside *basicThing
	app := fx.New(
		App(
			Provide(func() *basicThing { return &basicThing{value: "outside"} }),
			Populate(&outside),
			ReplaceBefore(&basicThing{value: "before-outside"}),
			Module("child",
				Provide(func() *basicThing { return &basicThing{value: "inside"} }, Name("inside")),
				Populate(&inside, Name("inside")),
			),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if outside == nil || outside.value != "before-outside" {
		t.Fatalf("unexpected outside: %#v", outside)
	}
	if inside == nil || inside.value != "inside" {
		t.Fatalf("unexpected inside: %#v", inside)
	}
}

func TestReplaceAfterRejectsGroup(t *testing.T) {
	app := fx.New(
		App(
			ReplaceAfter(depThing{id: 99}, Group("deps")),
			Supply(depThing{id: 1}, Group("deps")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err == nil {
		_ = app.Stop(ctx)
		t.Fatal("expected start to fail")
	}
}
