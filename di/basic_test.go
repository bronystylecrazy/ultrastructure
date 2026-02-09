package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestProvideAndSupplyBasic(t *testing.T) {
	var provided *basicThing
	var supplied *basicThing
	app := fxtest.New(t,
		App(
			Provide(newBasicThing),
			Supply(&basicThing{value: "supplied"}, Name("supplied")),
			Populate(&provided),
			Populate(&supplied, Name("supplied")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if provided == nil || provided.value != "provided" {
		t.Fatalf("unexpected provided: %#v", provided)
	}
	if supplied == nil || supplied.value != "supplied" {
		t.Fatalf("unexpected supplied: %#v", supplied)
	}
}

func TestDefaultProvidesWhenMissing(t *testing.T) {
	var got *basicThing
	app := fxtest.New(t,
		App(
			Default(&basicThing{value: "default"}),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "default" {
		t.Fatalf("unexpected default value: %#v", got)
	}
}

func TestParamsGroupTagOnProvide(t *testing.T) {
	var collector *depCollector
	app := fxtest.New(t,
		App(
			Supply(depThing{id: 1}, Group("deps")),
			Supply(depThing{id: 2}, Group("deps")),
			Provide(newDepCollector, Params(Group("deps"))),
			Populate(&collector),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if collector == nil || len(collector.ids) != 2 {
		t.Fatalf("unexpected collector: %#v", collector)
	}
}
