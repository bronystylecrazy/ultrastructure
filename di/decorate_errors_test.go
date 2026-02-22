package di

import (
	"strings"
	"testing"
)

type decoGroupDep interface {
	ID() int
}

type decoGroupDepImpl struct {
	id int
}

func (g *decoGroupDepImpl) ID() int { return g.id }

func startAppError(t *testing.T, nodes ...any) error {
	t.Helper()
	app := NewFxtestAppAllowErr(t, App(nodes...).Build())
	if err := app.Start(t.Context()); err != nil {
		return err
	}
	_ = app.Stop(t.Context())
	return nil
}

func TestDecorateRejectsNonFunction(t *testing.T) {
	err := startAppError(t,
		Provide(newBasicThing),
		Decorate(123),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errDecorateFunctionRequired) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsNoArgs(t *testing.T) {
	err := startAppError(t,
		Provide(newBasicThing),
		Decorate(func() *basicThing { return &basicThing{value: "x"} }),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errDecorateTooFewArgs) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsReturnCount(t *testing.T) {
	err := startAppError(t,
		Provide(func() *decoGroupDepImpl { return &decoGroupDepImpl{id: 1} }, As[decoGroupDep](`group:"items"`)),
		Decorate(func(decoGroupDep) {}, Group("items")),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errDecorateReturnCount) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsSecondReturnType(t *testing.T) {
	err := startAppError(t,
		Provide(func() *decoGroupDepImpl { return &decoGroupDepImpl{id: 1} }, As[decoGroupDep](`group:"items"`)),
		Decorate(func(d decoGroupDep) (decoGroupDep, string) { return d, "nope" }, Group("items")),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errDecorateSecondResult) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsMultipleResultTags(t *testing.T) {
	err := startAppError(t,
		Provide(newBasicThing),
		Decorate(
			func(b *basicThing) *basicThing { return b },
			Name("primary"),
			Name("secondary"),
		),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errDecorateNameGroupSingle) {
		t.Fatalf("unexpected error: %v", err)
	}
}

type badDecorateResultTag struct{}

func (badDecorateResultTag) applyBind(*bindConfig) {}
func (badDecorateResultTag) applyParam(cfg *paramConfig) {
	cfg.resultTags = append(cfg.resultTags, `foo:"bar"`)
}

func TestDecorateRejectsUnsupportedResultTag(t *testing.T) {
	err := startAppError(t,
		Provide(newBasicThing),
		Decorate(
			func(b *basicThing) *basicThing { return b },
			badDecorateResultTag{},
		),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "unsupported decorate tag") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecorateRejectsParamsCountMismatch(t *testing.T) {
	err := startAppError(t,
		Provide(newBasicThing),
		Supply(&decoDep{val: "one"}, Name("one")),
		Supply(&decoDep{val: "two"}, Name("two")),
		Decorate(func(b *basicThing, one *decoDep, two *decoDep) *basicThing { return b }, Params(Name("one"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "decorate_errors_test.go:") {
		t.Fatalf("expected source location in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Params count must match decorate function parameter count") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "di wiring:") {
		t.Fatalf("expected wiring location marker, got: %v", err)
	}
}

func TestDecorateRejectsVariadicParamsCountMismatch(t *testing.T) {
	err := startAppError(t,
		Provide(newBasicThing),
		Supply(&decoDep{val: "one"}, Name("one")),
		Supply(&decoDep{val: "two"}, Name("two")),
		Decorate(func(b *basicThing, rest ...*decoDep) *basicThing { return b }, Params(Name("one"), Name("two"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "decorate_errors_test.go:") {
		t.Fatalf("expected source location in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "expected 1, got 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}
