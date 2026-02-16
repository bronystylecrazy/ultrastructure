package us

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestNewAcceptsNodeSlice(t *testing.T) {
	var got string
	nodes := []di.Node{
		di.Supply("value"),
		di.Populate(&got),
	}

	app := fxtest.New(t, New(nodes).Build())
	defer app.RequireStart().RequireStop()

	if got != "value" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestNewRun(t *testing.T) {
	nodes := []di.Node{
		di.Invoke(func(shutdowner fx.Shutdowner) {
			_ = shutdowner.Shutdown()
		}),
	}

	if err := New(nodes).Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestNewProvidesDefaultModules(t *testing.T) {
	var root *cmd.Root
	var router fiber.Router

	app := fxtest.New(t,
		New(
			di.Populate(&root),
			di.Populate(&router),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if root == nil {
		t.Fatal("expected cmd root from default modules")
	}
	if router == nil {
		t.Fatal("expected fiber router from default modules")
	}
}
