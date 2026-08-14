package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

type scopedSource struct{ v string }
type scopedConsumer struct{ from string }

func newScopedSource() *scopedSource                    { return &scopedSource{v: "source"} }
func newScopedConsumer(s *scopedSource) *scopedConsumer { return &scopedConsumer{from: s.v} }

func resolveScopedConsumer(t *testing.T, nodes ...any) string {
	t.Helper()
	var got *scopedConsumer
	app := fxtest.New(t, App(append(nodes, Populate(&got))...).Build())
	app.RequireStart().RequireStop()
	if got == nil {
		return "<nil>"
	}
	return got.from
}

func TestDefaultConstructorResolvesAnotherDefault(t *testing.T) {
	if v := resolveScopedConsumer(t, Default(newScopedSource), Default(newScopedConsumer)); v != "source" {
		t.Fatalf("got %q want %q", v, "source")
	}
}

func TestDefaultConstructorResolvesAProvide(t *testing.T) {
	if v := resolveScopedConsumer(t, Provide(newScopedSource), Default(newScopedConsumer)); v != "source" {
		t.Fatalf("got %q want %q", v, "source")
	}
}

// Options group nodes without opening a scope, so a default declared inside one
// must still bind for consumers written outside it.
func TestDefaultInsideOptionsBindsForOuterConsumer(t *testing.T) {
	if v := resolveScopedConsumer(t, Options(Default(newScopedSource), Default(newScopedConsumer))); v != "source" {
		t.Fatalf("got %q want %q", v, "source")
	}
}

func TestOuterReplaceOutranksDefaultInsideOptions(t *testing.T) {
	v := resolveScopedConsumer(t,
		Replace(&scopedSource{v: "replaced"}),
		Options(Default(newScopedSource), Default(newScopedConsumer)),
	)
	if v != "replaced" {
		t.Fatalf("got %q want %q", v, "replaced")
	}
}
