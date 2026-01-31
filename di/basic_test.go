package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

func TestProvideAndSupplyBasic(t *testing.T) {
	var provided *basicThing
	var supplied *basicThing
	app := fx.New(
		App(
			Provide(newBasicThing),
			Supply(&basicThing{value: "supplied"}, Name("supplied")),
			Populate(&provided),
			Populate(&supplied, Name("supplied")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if provided == nil || provided.value != "provided" {
		t.Fatalf("unexpected provided: %#v", provided)
	}
	if supplied == nil || supplied.value != "supplied" {
		t.Fatalf("unexpected supplied: %#v", supplied)
	}
}

func TestDefaultProvidesWhenMissing(t *testing.T) {
	var got *basicThing
	app := fx.New(
		App(
			Default(&basicThing{value: "default"}),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got == nil || got.value != "default" {
		t.Fatalf("unexpected default value: %#v", got)
	}
}

func TestParamsGroupTagOnProvide(t *testing.T) {
	var collector *depCollector
	app := fx.New(
		App(
			Supply(depThing{id: 1}, Group("deps")),
			Supply(depThing{id: 2}, Group("deps")),
			Provide(newDepCollector, Params(Group("deps"))),
			Populate(&collector),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if collector == nil || len(collector.ids) != 2 {
		t.Fatalf("unexpected collector: %#v", collector)
	}
}
