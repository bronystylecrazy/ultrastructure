package ustest

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
)

func TestNewBuildsUSApp(t *testing.T) {
	var got string
	app := New(t,
		di.Supply("ok"),
		di.Populate(&got),
	)
	defer app.RequireStart().RequireStop()

	if got != "ok" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestRequiredStopAlias(t *testing.T) {
	app := New(t)
	defer app.RequireStart().RequireStop()
}
