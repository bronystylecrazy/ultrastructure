package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

type precedenceThing struct{ value string }

func newPrecedenceProvide() *precedenceThing { return &precedenceThing{value: "provide"} }
func newPrecedenceDefault() *precedenceThing { return &precedenceThing{value: "default"} }

func resolvePrecedenceThing(t *testing.T, nodes ...any) string {
	t.Helper()
	var got *precedenceThing
	app := fxtest.New(t, App(append(nodes, Populate(&got))...).Build())
	app.RequireStart().RequireStop()
	if got == nil {
		return "<nil>"
	}
	return got.value
}

func TestDefaultBindsWhenNothingElseDoes(t *testing.T) {
	if v := resolvePrecedenceThing(t, Default(newPrecedenceDefault)); v != "default" {
		t.Fatalf("got %q want %q", v, "default")
	}
}

func TestProvideOutranksDefaultInEitherOrder(t *testing.T) {
	if v := resolvePrecedenceThing(t, Provide(newPrecedenceProvide), Default(newPrecedenceDefault)); v != "provide" {
		t.Fatalf("provide first: got %q want %q", v, "provide")
	}
	if v := resolvePrecedenceThing(t, Default(newPrecedenceDefault), Provide(newPrecedenceProvide)); v != "provide" {
		t.Fatalf("default first: got %q want %q", v, "provide")
	}
}

func TestReplaceOutranksDefaultInEitherOrder(t *testing.T) {
	replaced := &precedenceThing{value: "replaced"}
	if v := resolvePrecedenceThing(t, Default(newPrecedenceDefault), Replace(replaced)); v != "replaced" {
		t.Fatalf("default first: got %q want %q", v, "replaced")
	}
	if v := resolvePrecedenceThing(t, Replace(replaced), Default(newPrecedenceDefault)); v != "replaced" {
		t.Fatalf("replace first: got %q want %q", v, "replaced")
	}
}

func TestReplaceOutranksDefaultWithProvidePresent(t *testing.T) {
	replaced := &precedenceThing{value: "replaced"}
	v := resolvePrecedenceThing(t, Default(newPrecedenceDefault), Provide(newPrecedenceProvide), Replace(replaced))
	if v != "replaced" {
		t.Fatalf("got %q want %q", v, "replaced")
	}
}

func TestDefaultInNestedScopeYieldsToOuterProvide(t *testing.T) {
	if v := resolvePrecedenceThing(t, Options(Default(newPrecedenceDefault)), Provide(newPrecedenceProvide)); v != "provide" {
		t.Fatalf("got %q want %q", v, "provide")
	}
}
