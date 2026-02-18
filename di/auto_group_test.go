package di

import (
	"reflect"
	"testing"

	"go.uber.org/fx/fxtest"
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
	app := fxtest.New(t,
		App(
			AutoGroup[autoGroupThing](),
			Provide(newAutoGroupThing),
			Populate(&things, Group("autogroupthing")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(things))
	}
	if things[0].Name() != "ok" {
		t.Fatalf("unexpected thing name %q", things[0].Name())
	}
}

func TestAutoGroupIgnore(t *testing.T) {
	var things []autoGroupThing
	app := fxtest.New(t,
		App(
			AutoGroup[autoGroupThing](),
			Provide(newAutoGroupThing, AutoGroupIgnore()),
			Populate(&things, Group("autogroupthing")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(things) != 0 {
		t.Fatalf("expected 0 things, got %d", len(things))
	}
}

func TestAutoGroupModuleInheritance(t *testing.T) {
	var things []autoGroupThing
	app := fxtest.New(t,
		App(
			AutoGroup[autoGroupThing]("things"),
			Module("child",
				Provide(newAutoGroupThing),
			),
			Populate(&things, Group("things")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(things))
	}
	if things[0].Name() != "ok" {
		t.Fatalf("unexpected thing name %q", things[0].Name())
	}
}

func TestAutoGroupModuleLocalRule(t *testing.T) {
	var things []autoGroupThing
	app := fxtest.New(t,
		App(
			Module("child",
				AutoGroup[autoGroupThing]("child-things"),
				Provide(newAutoGroupThing),
			),
			Populate(&things, Group("child-things")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(things))
	}
	if things[0].Name() != "ok" {
		t.Fatalf("unexpected thing name %q", things[0].Name())
	}
}

func TestAutoGroupModuleDoesNotLeakToParent(t *testing.T) {
	var things []autoGroupThing
	app := fxtest.New(t,
		App(
			Module("child",
				AutoGroup[autoGroupThing]("child-things"),
				Provide(newAutoGroupThing),
			),
			Provide(newAutoGroupThingAlt),
			Populate(&things, Group("child-things")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(things))
	}
	if things[0].Name() != "ok" {
		t.Fatalf("unexpected thing name %q", things[0].Name())
	}
}

func TestAutoGroupModuleOverrideGroup(t *testing.T) {
	var things []autoGroupThing
	app := fxtest.New(t,
		App(
			AutoGroup[autoGroupThing]("parent-things"),
			Module("child",
				AutoGroup[autoGroupThing]("child-things"),
				Provide(newAutoGroupThing),
			),
			Populate(&things, Group("child-things")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(things))
	}
	if things[0].Name() != "ok" {
		t.Fatalf("unexpected thing name %q", things[0].Name())
	}
}

func TestAutoGroupFilterOption(t *testing.T) {
	cfg := bindConfig{
		autoGroups: []autoGroupRule{{}},
	}
	opt := AutoGroupFilter(func(reflect.Type) bool { return false })
	opt.applyBind(&cfg)
	if cfg.err != nil {
		t.Fatalf("unexpected bind error: %v", cfg.err)
	}
	if cfg.autoGroups[0].filter == nil {
		t.Fatalf("expected filter override to be applied")
	}
	if cfg.autoGroups[0].filter(reflect.TypeOf(&autoGroupThingImpl{})) {
		t.Fatalf("expected filter function to return false")
	}
}

func TestAutoGroupAsSelfOption(t *testing.T) {
	cfg := bindConfig{
		autoGroups: []autoGroupRule{{}},
	}
	opt := AutoGroupAsSelf()
	opt.applyBind(&cfg)
	if cfg.err != nil {
		t.Fatalf("unexpected bind error: %v", cfg.err)
	}
	if !cfg.autoGroups[0].asSelf {
		t.Fatalf("expected asSelf override to be applied")
	}
}
