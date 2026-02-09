package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestReplaceOverridesProvide(t *testing.T) {
	var got *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing),
			Replace(&basicThing{value: "replaced"}),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

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
	app := fxtest.New(t,
		App(
			Provide(func() *firstThing { return &firstThing{value: "orig"} }),
			Populate(&first),
			ReplaceBefore(&firstThing{value: "before"}),
			ReplaceAfter(&secondThing{value: "after"}),
			Provide(func() *secondThing { return &secondThing{value: "later"} }),
			Populate(&second),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

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
	app := fxtest.New(t,
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
	defer app.RequireStart().RequireStop()

	if outside == nil || outside.value != "before-outside" {
		t.Fatalf("unexpected outside: %#v", outside)
	}
	if inside == nil || inside.value != "inside" {
		t.Fatalf("unexpected inside: %#v", inside)
	}
}

func TestReplaceAfterRejectsGroup(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			ReplaceAfter(depThing{id: 99}, Group("deps")),
			Supply(depThing{id: 1}, Group("deps")),
		).Build(),
	)
	if err := app.Start(t.Context()); err == nil {
		_ = app.Stop(t.Context())
		t.Fatal("expected start to fail")
	}
}
