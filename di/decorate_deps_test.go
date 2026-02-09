package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

type decoDep struct {
	val string
}

func TestDecorateWithExtraDepsAndTags(t *testing.T) {
	var got *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing),
			Supply(&decoDep{val: "dep"}, Name("dep")),
			Decorate(func(b *basicThing, dep *decoDep) *basicThing {
				b.value = b.value + "-" + dep.val
				return b
			}, Params(Name("dep"))),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided-dep" {
		t.Fatalf("unexpected decorated value: %#v", got)
	}
}

func TestDecorateGroupWithExtraDeps(t *testing.T) {
	var got []depThing
	app := fxtest.New(t,
		App(
			Supply(depThing{id: 1}, Group("deps")),
			Supply(depThing{id: 2}, Group("deps")),
			Supply(&decoDep{val: "dep"}, Name("dep")),
			Decorate(func(items []depThing, dep *decoDep) []depThing {
				if dep.val != "dep" {
					return items
				}
				for i := range items {
					items[i].id += 10
				}
				return items
			}, Group("deps"), Params(Name("dep"))),
			Populate(&got, Group("deps")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	seen := map[int]bool{}
	for _, item := range got {
		seen[item.id] = true
	}
	if !seen[11] || !seen[12] {
		t.Fatalf("unexpected items: %#v", got)
	}
}

func TestDecorateWithManyDeps(t *testing.T) {
	var got *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing),
			Supply(&decoDep{val: "one"}, Name("one")),
			Supply(&decoDep{val: "two"}, Name("two")),
			Decorate(func(b *basicThing, one *decoDep, two *decoDep) *basicThing {
				b.value = b.value + "-" + one.val + "-" + two.val
				return b
			}, Params(Name("one"), Name("two"))),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided-one-two" {
		t.Fatalf("unexpected decorated value: %#v", got)
	}
}

func TestDecorateNamedWithDeps(t *testing.T) {
	var primary *basicThing
	var secondary *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing, Name("primary")),
			Provide(newBasicThing, Name("secondary")),
			Supply(&decoDep{val: "dep"}, Name("dep")),
			Decorate(func(b *basicThing, dep *decoDep) *basicThing {
				b.value = b.value + "-" + dep.val
				return b
			}, Name("primary"), Params(Name("dep"))),
			Populate(&primary, Name("primary")),
			Populate(&secondary, Name("secondary")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if primary == nil || primary.value != "provided-dep" {
		t.Fatalf("unexpected primary: %#v", primary)
	}
	if secondary == nil || secondary.value != "provided" {
		t.Fatalf("unexpected secondary: %#v", secondary)
	}
}

func TestDecorateMultipleWithDifferentDeps(t *testing.T) {
	var got *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing),
			Supply(&decoDep{val: "one"}, Name("one")),
			Supply(&decoDep{val: "two"}, Name("two")),
			Decorate(func(b *basicThing, dep *decoDep) *basicThing {
				b.value = b.value + "-" + dep.val
				return b
			}, Params(Name("one"))),
			Decorate(func(b *basicThing, dep *decoDep) *basicThing {
				b.value = b.value + "-" + dep.val
				return b
			}, Params(Name("two"))),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "provided-one-two" {
		t.Fatalf("unexpected decorated value: %#v", got)
	}
}
