package di

import (
	"errors"
	"strings"
	"testing"
)

func startProvideAppError(t *testing.T, nodes ...any) error {
	t.Helper()
	app := NewFxtestAppAllowErr(t, App(nodes...).Build())
	if err := app.Start(t.Context()); err != nil {
		return err
	}
	_ = app.Stop(t.Context())
	return nil
}

func TestProvideRejectsNilConstructor(t *testing.T) {
	err := startProvideAppError(t, Provide(nil))
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errCannotInferType) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsNonFunction(t *testing.T) {
	err := startProvideAppError(t, Provide(123))
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errConstructorMustBeFunction) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsInvalidReturnCount(t *testing.T) {
	err := startProvideAppError(t, Provide(func() {}))
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errConstructorReturnCount) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsInvalidSecondResult(t *testing.T) {
	err := startProvideAppError(t, Provide(func() (*basicThing, string) { return nil, "nope" }))
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errConstructorSecondResult) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsWithTagsMixedAs(t *testing.T) {
	err := startProvideAppError(t,
		Provide(newBasicThing, Name("tagged"), As[*basicThing](), As[basicThing]()),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errWithTagsSingleAs) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSupplyRejectsNilValue(t *testing.T) {
	err := startProvideAppError(t, Supply(nil))
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errCannotInferType) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSupplyRejectsErrorValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = startProvideAppError(t, Supply(errors.New("nope")))
}

func TestSupplyRejectsParams(t *testing.T) {
	err := startProvideAppError(t,
		Supply(&basicThing{value: "x"}, Params(`name:"dep"`)),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errParamsNotSupportedWithSupply) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsEmptyName(t *testing.T) {
	err := startProvideAppError(t, Provide(newBasicThing, Name("")))
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errNameEmpty) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsEmptyGroup(t *testing.T) {
	err := startProvideAppError(t, Provide(newBasicThing, Group("")))
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errGroupNameEmpty) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsParamsCountMismatch(t *testing.T) {
	err := startProvideAppError(t,
		Provide(func(a *basicThing, b *basicThing) *basicThing {
			return a
		}, Params(Name("one"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "provide_errors_test.go:") {
		t.Fatalf("expected source location in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Params count must match constructor parameter count") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideRejectsVariadicParamsCountMismatch(t *testing.T) {
	err := startProvideAppError(t,
		Provide(func(a *basicThing, rest ...*basicThing) *basicThing {
			return a
		}, Params(Name("one"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "provide_errors_test.go:") {
		t.Fatalf("expected source location in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "expected 2, got 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
