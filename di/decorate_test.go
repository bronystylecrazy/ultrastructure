package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

type testHandler interface {
	ID() string
}

type testHandlerImpl struct {
	id string
}

func (h *testHandlerImpl) ID() string { return h.id }

func newTestHandler() *testHandlerImpl {
	return &testHandlerImpl{id: "raw"}
}

func TestDecorateAutoGroupSeesDecoratedInstance(t *testing.T) {
	var handlers []testHandler
	app := fxtest.New(t,
		App(
			AutoGroup[testHandler]("handlers"),
			Provide(newTestHandler),
			Decorate(func(h *testHandlerImpl) *testHandlerImpl {
				h.id = "decorated"
				return h
			}),
			Populate(&handlers, Group("handlers")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(handlers))
	}
	if handlers[0].ID() != "decorated" {
		t.Fatalf("expected decorated handler, got %q", handlers[0].ID())
	}
}

func TestDecorateInvalidSignatureFails(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Provide(newBasicThing),
			Decorate(func() *basicThing { return &basicThing{} }),
			Decorate(func() *basicThing { return &basicThing{} }),
		).Build(),
	)
	if err := app.Start(t.Context()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAutoGroupIgnoreTypeOption(t *testing.T) {
	var handlers []testHandler
	app := fxtest.New(t,
		App(
			AutoGroup[testHandler]("handlers"),
			Provide(newTestHandler, AutoGroupIgnoreType[testHandler]("handlers")),
			Populate(&handlers, Group("handlers")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if len(handlers) != 0 {
		t.Fatalf("expected 0 handlers, got %d", len(handlers))
	}
}
