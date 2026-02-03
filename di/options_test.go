package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

func TestOptionsAsNodeAndOption(t *testing.T) {
	var named string
	var grouped []string
	var invoked string

	app := fx.New(
		App(
			Provide(func() string { return "value" },
				Options(
					Name("primary"),
					Group("items"),
				),
			),
			Invoke(
				func(s string) {
					invoked = s
				},
				Options(Name("primary")),
			),
			Populate(&named, Name("primary")),
			Populate(&grouped, Group("items")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if named != "value" {
		t.Fatalf("unexpected named: %q", named)
	}
	if invoked != "value" {
		t.Fatalf("unexpected invoked: %q", invoked)
	}
	if len(grouped) != 1 || grouped[0] != "value" {
		t.Fatalf("unexpected grouped: %#v", grouped)
	}
}
