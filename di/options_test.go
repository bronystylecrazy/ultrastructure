package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestOptionsAsNodeAndOption(t *testing.T) {
	var named string
	var grouped []string
	var invoked string

	app := fxtest.New(t,
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
	defer app.RequireStart().RequireStop()

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
