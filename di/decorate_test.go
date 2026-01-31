package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
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
	app := fx.New(
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(handlers))
	}
	if handlers[0].ID() != "decorated" {
		t.Fatalf("expected decorated handler, got %q", handlers[0].ID())
	}
}
