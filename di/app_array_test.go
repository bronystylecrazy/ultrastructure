package di

import (
	"testing"

	"go.uber.org/fx/fxtest"
)

func TestAppAcceptsNodeSlice(t *testing.T) {
	var got string
	nodes := []Node{
		Supply("value"),
		Populate(&got),
	}

	app := fxtest.New(t, App(nodes).Build())
	defer app.RequireStart().RequireStop()

	if got != "value" {
		t.Fatalf("unexpected value: %q", got)
	}
}
