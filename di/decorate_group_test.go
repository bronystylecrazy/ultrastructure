package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

func TestDecorateWithNameTag(t *testing.T) {
	var primary *namedThing
	var secondary *namedThing
	app := fx.New(
		App(
			Provide(newNamedPrimary, Name("primary")),
			Provide(newNamedSecondary, Name("secondary")),
			Decorate(func(n *namedThing) *namedThing {
				n.value = n.value + "-decorated"
				return n
			}, Name("primary")),
			Populate(&primary, Name("primary")),
			Populate(&secondary, Name("secondary")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if primary == nil || primary.value != "primary-decorated" {
		t.Fatalf("unexpected primary: %#v", primary)
	}
	if secondary == nil || secondary.value != "secondary" {
		t.Fatalf("unexpected secondary: %#v", secondary)
	}
}

func TestDecorateGroupSlice(t *testing.T) {
	var got []depThing
	app := fx.New(
		App(
			Supply(depThing{id: 1}, Group("deps")),
			Supply(depThing{id: 2}, Group("deps")),
			Decorate(func(items []depThing) []depThing {
				for i := range items {
					items[i].id *= 10
				}
				return items
			}, Group("deps")),
			Populate(&got, Group("deps")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	seen := map[int]bool{}
	for _, item := range got {
		seen[item.id] = true
	}
	if !seen[10] || !seen[20] {
		t.Fatalf("unexpected items: %#v", got)
	}
}
