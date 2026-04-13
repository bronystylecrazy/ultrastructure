package otel

import (
	"reflect"
	"testing"
)

func TestParseListParsesJSONArray(t *testing.T) {
	got := ParseList(`["layer:web","logger:sqlc","file:web/fiber.go:120-180"]`)
	want := []string{"layer:web", "logger:sqlc", "file:web/fiber.go:120-180"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected list: got %#v want %#v", got, want)
	}
}

func TestParseListFallsBackToCommaSeparated(t *testing.T) {
	got := ParseList(`layer:web, logger:sqlc, file:web/fiber.go:120-180`)
	want := []string{"layer:web", "logger:sqlc", "file:web/fiber.go:120-180"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected list: got %#v want %#v", got, want)
	}
}

func TestParseListFallsBackWhenJSONIsInvalid(t *testing.T) {
	got := ParseList(`[layer:web],logger:sqlc`)
	want := []string{"[layer:web]", "logger:sqlc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected list: got %#v want %#v", got, want)
	}
}
