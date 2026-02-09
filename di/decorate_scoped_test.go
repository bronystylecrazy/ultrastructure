package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestProvideScopedDecorateOnlyTargetsProvider(t *testing.T) {
	var primary *basicThing
	var secondary *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing, Name("primary"), Decorate(func(b *basicThing) *basicThing {
				b.value = b.value + "-decorated"
				return b
			})),
			Provide(func() *basicThing { return &basicThing{value: "secondary"} }, Name("secondary")),
			Populate(&primary, Name("primary")),
			Populate(&secondary, Name("secondary")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if primary == nil || primary.value != "provided-decorated" {
		t.Fatalf("unexpected primary: %#v", primary)
	}
	if secondary == nil || secondary.value != "secondary" {
		t.Fatalf("unexpected secondary: %#v", secondary)
	}
}

func TestProvideScopedDecorateGroupOnlyTargetsGroup(t *testing.T) {
	var deps []depThing
	var others []depThing
	app := fxtest.New(t,
		App(
			Supply(depThing{id: 1}, Group("deps"), Decorate(func(items []depThing) []depThing {
				for i := range items {
					items[i].id *= 10
				}
				return items
			})),
			Supply(depThing{id: 2}, Group("other")),
			Populate(&deps, Group("deps")),
			Populate(&others, Group("other")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(deps) != 1 || deps[0].id != 10 {
		t.Fatalf("unexpected deps: %#v", deps)
	}
	if len(others) != 1 || others[0].id != 2 {
		t.Fatalf("unexpected others: %#v", others)
	}
}

type scopedHandler interface {
	ID() string
}

type scopedHandlerImpl struct {
	id string
}

func (s *scopedHandlerImpl) ID() string { return s.id }

func newScopedHandler() *scopedHandlerImpl {
	return &scopedHandlerImpl{id: "scoped"}
}

type altHandler interface {
	Name() string
}

type altHandlerImpl struct {
	name string
}

func (a *altHandlerImpl) Name() string { return a.name }

func newAltHandler() *altHandlerImpl {
	return &altHandlerImpl{name: "alt"}
}

func TestProvideScopedDecorateWithAutoGroup(t *testing.T) {
	var handlers []scopedHandler
	var alts []altHandler
	app := fxtest.New(t,
		App(
			AutoGroup[scopedHandler]("handlers"),
			AutoGroup[altHandler]("alt-handlers"),
			Provide(newScopedHandler, Decorate(func(h *scopedHandlerImpl) *scopedHandlerImpl {
				h.id = h.id + "-decorated"
				return h
			})),
			Provide(newAltHandler),
			Populate(&handlers, Group("handlers")),
			Populate(&alts, Group("alt-handlers")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(handlers) != 1 || handlers[0].ID() != "scoped-decorated" {
		t.Fatalf("unexpected handlers: %#v", handlers)
	}
	if len(alts) != 1 || alts[0].Name() != "alt" {
		t.Fatalf("unexpected alt handlers: %#v", alts)
	}
}

func TestModuleScopedDecorateAutoGroupAffectsAllProviders(t *testing.T) {
	var handlers []scopedHandler
	app := fxtest.New(t,
		App(
			Module("handlers",
				AutoGroup[scopedHandler]("handlers"),
				Provide(func() *scopedHandlerImpl { return &scopedHandlerImpl{id: "one"} }, Name("one")),
				Provide(func() *scopedHandlerImpl { return &scopedHandlerImpl{id: "two"} }, Name("two")),
				Decorate(func(items []scopedHandler) []scopedHandler {
					for i, h := range items {
						impl, ok := h.(*scopedHandlerImpl)
						if !ok {
							continue
						}
						impl.id = impl.id + "-decorated"
						items[i] = impl
					}
					return items
				}, Group("handlers")),
				Populate(&handlers, Group("handlers")),
			),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(handlers) != 2 {
		t.Fatalf("unexpected handlers: %#v", handlers)
	}
	seen := map[string]bool{}
	for _, h := range handlers {
		seen[h.ID()] = true
	}
	if !seen["one-decorated"] || !seen["two-decorated"] {
		t.Fatalf("unexpected handlers: %#v", handlers)
	}
}
