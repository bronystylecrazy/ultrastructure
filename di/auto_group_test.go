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

type autoGroupThingAlt struct {
	name string
}

func (t *autoGroupThingAlt) Name() string { return t.name }

func newAutoGroupThingAlt() *autoGroupThingAlt {
	return &autoGroupThingAlt{name: "alt"}
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

func TestAutoGroupModuleInheritance(t *testing.T) {
	var things []autoGroupThing
	app := fx.New(
		App(
			AutoGroup[autoGroupThing]("things"),
			Module("child",
				Provide(newAutoGroupThing),
			),
			Populate(&things, Group("things")),
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

func TestAutoGroupModuleLocalRule(t *testing.T) {
	var things []autoGroupThing
	app := fx.New(
		App(
			Module("child",
				AutoGroup[autoGroupThing]("child-things"),
				Provide(newAutoGroupThing),
			),
			Populate(&things, Group("child-things")),
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

func TestAutoGroupModuleDoesNotLeakToParent(t *testing.T) {
	var things []autoGroupThing
	app := fx.New(
		App(
			Module("child",
				AutoGroup[autoGroupThing]("child-things"),
				Provide(newAutoGroupThing),
			),
			Provide(newAutoGroupThingAlt),
			Populate(&things, Group("child-things")),
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

func TestAutoGroupModuleOverrideGroup(t *testing.T) {
	var things []autoGroupThing
	app := fx.New(
		App(
			AutoGroup[autoGroupThing]("parent-things"),
			Module("child",
				AutoGroup[autoGroupThing]("child-things"),
				Provide(newAutoGroupThing),
			),
			Populate(&things, Group("child-things")),
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
