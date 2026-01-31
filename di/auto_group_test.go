package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

type autoGroupThing interface {
	Name() string
}

type autoGroupThingImpl struct {
	name string
}

func (t *autoGroupThingImpl) Name() string { return t.name }

func newAutoGroupThing() *autoGroupThingImpl {
	return &autoGroupThingImpl{name: "ok"}
}

func TestAutoGroupDefaultName(t *testing.T) {
	var things []autoGroupThing
	app := fx.New(
		App(
			AutoGroup[autoGroupThing](),
			Provide(newAutoGroupThing),
			Populate(&things, Group("autogroupthing")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if len(things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(things))
	}
	if things[0].Name() != "ok" {
		t.Fatalf("unexpected thing name %q", things[0].Name())
	}
}

func TestAutoGroupIgnore(t *testing.T) {
	var things []autoGroupThing
	app := fx.New(
		App(
			AutoGroup[autoGroupThing](),
			Provide(newAutoGroupThing, AutoGroupIgnore()),
			Populate(&things, Group("autogroupthing")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if len(things) != 0 {
		t.Fatalf("expected 0 things, got %d", len(things))
	}
}
