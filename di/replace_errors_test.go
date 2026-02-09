package di

import (
	"strings"
	"testing"
)

func TestReplaceRejectsNameOrGroup(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Replace(&basicThing{value: "x"}, Name("named")),
		).Build(),
	)
	if err := app.Start(t.Context()); err == nil {
		_ = app.Stop(t.Context())
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errReplaceNoNamedOrGroupedExports) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplaceRejectsPrivate(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Replace(&basicThing{value: "x"}, Private()),
		).Build(),
	)
	if err := app.Start(t.Context()); err == nil {
		_ = app.Stop(t.Context())
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errReplaceNoPrivatePublic) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplaceRejectsDecorateOption(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Replace(&basicThing{value: "x"}, Decorate(func(b *basicThing) *basicThing { return b })),
		).Build(),
	)
	if err := app.Start(t.Context()); err == nil {
		_ = app.Stop(t.Context())
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errReplaceNoDecorate) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultRejectsDecorateOption(t *testing.T) {
	app := NewFxtestAppAllowErr(t,
		App(
			Default(&basicThing{value: "x"}, Decorate(func(b *basicThing) *basicThing { return b })),
		).Build(),
	)
	if err := app.Start(t.Context()); err == nil {
		_ = app.Stop(t.Context())
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errDefaultNoDecorate) {
		t.Fatalf("unexpected error: %v", err)
	}
}
