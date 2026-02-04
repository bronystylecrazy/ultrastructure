package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

type priorityThing interface {
	Name() string
}

type priorityThingEarly struct {
	name string
}

func (t *priorityThingEarly) Name() string { return t.name }

type priorityThingNormal struct {
	name string
}

func (t *priorityThingNormal) Name() string { return t.name }

type priorityThingLater struct {
	name string
}

func (t *priorityThingLater) Name() string { return t.name }

func newPriorityThingEarly(name string) *priorityThingEarly {
	return &priorityThingEarly{name: name}
}

func newPriorityThingNormal(name string) *priorityThingNormal {
	return &priorityThingNormal{name: name}
}

func newPriorityThingLater(name string) *priorityThingLater {
	return &priorityThingLater{name: name}
}

func TestPriorityOrdersAutoGroup(t *testing.T) {
	var things []priorityThing
	app := fx.New(
		App(
			AutoGroup[priorityThing]("priority-things"),
			Provide(func() *priorityThingEarly { return newPriorityThingEarly("earliest") }, Priority(Earliest)),
			Provide(func() *priorityThingNormal { return newPriorityThingNormal("normal") }, Priority(Normal)),
			Provide(func() *priorityThingLater { return newPriorityThingLater("later") }, Priority(Later)),
			Populate(&things, Group("priority-things")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if len(things) != 3 {
		t.Fatalf("expected 3 things, got %d", len(things))
	}
	for _, item := range things {
		raw, ok := ReflectMetadataAny(item)
		if !ok {
			t.Fatalf("missing metadata for %q", item.Name())
		}
		meta, ok := raw.([]any)
		if !ok {
			t.Fatalf("unexpected metadata type for %q", item.Name())
		}
		hasPriority := false
		for _, m := range meta {
			if _, ok := m.(priorityOrder); ok {
				hasPriority = true
			}
		}
		if !hasPriority {
			t.Fatalf("missing priorityOrder for %q", item.Name())
		}
	}
	if things[0].Name() != "earliest" || things[1].Name() != "normal" || things[2].Name() != "later" {
		t.Fatalf("unexpected order: %q, %q, %q", things[0].Name(), things[1].Name(), things[2].Name())
	}
}

func TestBetweenPriority(t *testing.T) {
	mid := Between(Earlier, Later)
	if mid <= Earlier || mid >= Later {
		t.Fatalf("expected Between(Earlier, Later) to be between %d and %d, got %d", Earlier, Later, mid)
	}
}
