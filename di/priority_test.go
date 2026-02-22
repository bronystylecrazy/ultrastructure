package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
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
	app := fxtest.New(t,
		App(
			AutoGroup[priorityThing]("priority-things"),
			Provide(func() *priorityThingEarly { return newPriorityThingEarly("earliest") }, Priority(Earliest)),
			Provide(func() *priorityThingNormal { return newPriorityThingNormal("normal") }, Priority(Normal)),
			Provide(func() *priorityThingLater { return newPriorityThingLater("later") }, Priority(Later)),
			Populate(&things, Group("priority-things")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

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

func TestPriorityIndexUsesLastMetadataValue(t *testing.T) {
	value := &priorityThingNormal{name: "normal"}
	RegisterMetadata(value, priorityOrder{Index: int64(Earlier)}, priorityOrder{Index: int64(Later)})

	got, ok := PriorityIndex(value)
	if !ok {
		t.Fatalf("expected priority metadata to be found")
	}
	if got != int64(Later) {
		t.Fatalf("unexpected priority index: got %d want %d", got, int64(Later))
	}
}

func TestOrderIndexUsesLastMetadataValue(t *testing.T) {
	value := &priorityThingNormal{name: "normal"}
	RegisterMetadata(value, autoGroupOrder{Index: 1}, autoGroupOrder{Index: 7})

	got, ok := OrderIndex(value)
	if !ok {
		t.Fatalf("expected order metadata to be found")
	}
	if got != 7 {
		t.Fatalf("unexpected order index: got %d want %d", got, 7)
	}
}
