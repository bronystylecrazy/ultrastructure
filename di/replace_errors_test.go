package di

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx"
)

func TestReplaceRejectsNameOrGroup(t *testing.T) {
	app := fx.New(
		App(
			Replace(&basicThing{value: "x"}, Name("named")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err == nil {
		_ = app.Stop(ctx)
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errReplaceNoNamedOrGroupedExports) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplaceRejectsPrivate(t *testing.T) {
	app := fx.New(
		App(
			Replace(&basicThing{value: "x"}, Private()),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err == nil {
		_ = app.Stop(ctx)
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errReplaceNoPrivatePublic) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplaceRejectsDecorateOption(t *testing.T) {
	app := fx.New(
		App(
			Replace(&basicThing{value: "x"}, Decorate(func(b *basicThing) *basicThing { return b })),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err == nil {
		_ = app.Stop(ctx)
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errReplaceNoDecorate) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultRejectsDecorateOption(t *testing.T) {
	app := fx.New(
		App(
			Default(&basicThing{value: "x"}, Decorate(func(b *basicThing) *basicThing { return b })),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err == nil {
		_ = app.Stop(ctx)
		t.Fatal("expected start to fail")
	} else if !strings.Contains(err.Error(), errDefaultNoDecorate) {
		t.Fatalf("unexpected error: %v", err)
	}
}
