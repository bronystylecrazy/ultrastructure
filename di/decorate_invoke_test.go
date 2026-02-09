package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestDecorateRunsBeforeInvoke(t *testing.T) {
	var got string
	app := fxtest.New(t,
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
	defer app.RequireStart().RequireStop()

	if got != "provided-decorated" {
		t.Fatalf("unexpected invoke value: %q", got)
	}
}
