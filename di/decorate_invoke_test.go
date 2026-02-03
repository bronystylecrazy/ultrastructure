package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

func TestDecorateRunsBeforeInvoke(t *testing.T) {
	var got string
	app := fx.New(
		App(
			Provide(newBasicThing),
			Decorate(func(b *basicThing) *basicThing {
				b.value = b.value + "-decorated"
				return b
			}),
			Invoke(func(b *basicThing) {
				got = b.value
			}),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got != "provided-decorated" {
		t.Fatalf("unexpected invoke value: %q", got)
	}
}
