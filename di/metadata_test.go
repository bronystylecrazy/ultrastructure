package di

import (
	"reflect"
	"testing"

	"go.uber.org/fx/fxtest"
)

type metadataFanoutThing struct {
	id string
}

func TestMetadataGroupEmitsOneEntryPerProvider(t *testing.T) {
	var namedA *metadataFanoutThing
	var namedB *metadataFanoutThing
	var namedC *metadataFanoutThing
	var grouped []*metadataFanoutThing
	var metas []MetadataValue

	app := fxtest.New(t,
		App(
			Provide(
				func() *metadataFanoutThing { return &metadataFanoutThing{id: "provided"} },
				Metadata("provided-meta"),
				Name("a"),
				Name("b"),
				Group("items"),
			),
			Supply(
				&metadataFanoutThing{id: "supplied"},
				Metadata("supplied-meta"),
				Name("c"),
				Group("items"),
			),
			Populate(&namedA, Name("a")),
			Populate(&namedB, Name("b")),
			Populate(&namedC, Name("c")),
			Populate(&grouped, Group("items")),
			Populate(&metas, MetadataGroup()),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if namedA == nil || namedB == nil || namedC == nil {
		t.Fatalf("expected all named values to be populated")
	}
	if len(grouped) != 2 {
		t.Fatalf("expected 2 grouped values, got %d", len(grouped))
	}
	if len(metas) != 2 {
		t.Fatalf("expected 2 metadata values (one per provider), got %d", len(metas))
	}

	wantType := reflect.TypeOf(&metadataFanoutThing{})
	seen := map[string]bool{}
	for _, mv := range metas {
		if mv.Type != wantType {
			t.Fatalf("unexpected metadata type: got %v want %v", mv.Type, wantType)
		}
		values, ok := mv.Metadata.([]any)
		if !ok || len(values) != 1 {
			t.Fatalf("unexpected metadata payload: %#v", mv.Metadata)
		}
		s, ok := values[0].(string)
		if !ok {
			t.Fatalf("unexpected metadata value: %#v", values[0])
		}
		seen[s] = true
	}
	if !seen["provided-meta"] || !seen["supplied-meta"] {
		t.Fatalf("unexpected metadata tags: %#v", seen)
	}
}

