package di

import (
	"strconv"
	"strings"
	"testing"

	"go.uber.org/fx/fxtest"
)

type variadicArgA struct {
	val string
}

type variadicArgB struct {
	val string
}

type variadicWorker struct {
	val string
}

func TestProvideParamsVariadicHelperTagsVariadicParam(t *testing.T) {
	var got string

	app := fxtest.New(t,
		App(
			Supply(&variadicArgA{val: "a"}),
			Supply(&variadicArgA{val: "b"}, Name("second")),
			Supply(&variadicWorker{val: "w1"}, Group("workers")),
			Supply(&variadicWorker{val: "w2"}, Group("workers")),
			Provide(func(a *variadicArgA, b *variadicArgA, workers ...*variadicWorker) string {
				return a.val + ":" + b.val + ":" + strconv.Itoa(len(workers))
			}, Params(nil, Name("second"), Variadic(Group("workers")))),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:b:2" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestProvideVariadicRejectsMultipleTags(t *testing.T) {
	err := startProvideAppError(t,
		Supply(&basicThing{value: "one"}, Name("one")),
		Provide(func(a *basicThing, rest ...*basicThing) *basicThing {
			return a
		}, Params(Name("one"), Variadic(Name("two"), Group("workers")))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "Variadic expects exactly one parameter tag, got 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideVariadicTopLevelTagsLastVariadicParam(t *testing.T) {
	var got string

	app := fxtest.New(t,
		App(
			Supply(&variadicArgA{val: "a"}),
			Supply(&variadicArgB{val: "b"}),
			Supply(&variadicWorker{val: "w1"}, Group("workers")),
			Supply(&variadicWorker{val: "w2"}, Group("workers")),
			Provide(func(a *variadicArgA, b *variadicArgB, workers ...*variadicWorker) string {
				return a.val + ":" + b.val + ":" + strconv.Itoa(len(workers))
			}, Name("res"), Variadic(Group("workers"))),
			Invoke(func(s string) { got = s }, Params(Name("res"))),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:b:2" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestInvokeVariadicTopLevelTagsLastVariadicParam(t *testing.T) {
	var got string

	app := fxtest.New(t,
		App(
			Supply(&variadicArgA{val: "a"}),
			Supply(&variadicArgB{val: "b"}),
			Supply(&variadicWorker{val: "w1"}, Group("workers")),
			Supply(&variadicWorker{val: "w2"}, Group("workers")),
			Invoke(func(a *variadicArgA, b *variadicArgB, workers ...*variadicWorker) {
				got = a.val + ":" + b.val + ":" + strconv.Itoa(len(workers))
			}, Variadic(Group("workers"))),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:b:2" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestProvideVariadicTopLevelRejectsNonVariadicFunction(t *testing.T) {
	err := startProvideAppError(t,
		Supply(&basicThing{value: "one"}, Name("one")),
		Provide(func(a *basicThing) *basicThing {
			return a
		}, Variadic(Name("one"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errVariadicRequiresVariadicTarget) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideVariadicCannotMixWithParams(t *testing.T) {
	err := startProvideAppError(t,
		Supply(&basicThing{value: "one"}, Name("one")),
		Provide(func(a *basicThing, rest ...*basicThing) *basicThing {
			return a
		}, Params(Name("one"), Name("one")), Variadic(Group("workers"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), errVariadicWithParams) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProvideVariadicGroupHelper(t *testing.T) {
	var got string

	app := fxtest.New(t,
		App(
			Supply(&variadicArgA{val: "a"}),
			Supply(&variadicArgB{val: "b"}),
			Supply(&variadicWorker{val: "w1"}, Group("workers")),
			Supply(&variadicWorker{val: "w2"}, Group("workers")),
			Provide(func(a *variadicArgA, b *variadicArgB, workers ...*variadicWorker) string {
				return a.val + ":" + b.val + ":" + strconv.Itoa(len(workers))
			}, Name("res"), VariadicGroup("workers")),
			Invoke(func(s string) { got = s }, Params(Name("res"))),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:b:2" {
		t.Fatalf("unexpected value: %q", got)
	}
}
