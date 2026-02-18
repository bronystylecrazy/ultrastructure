package di

import (
	"strconv"
	"testing"

	"go.uber.org/fx/fxtest"
)

type skipArgA struct {
	val string
}

type skipArgB struct {
	val string
}

func TestInvokeParamsCanSkipFirstArgAndTagSecond(t *testing.T) {
	var got string
	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Supply(&skipArgB{val: "b"}, Name("second")),
			Invoke(func(a *skipArgA, b *skipArgB) {
				got = a.val + ":" + b.val
			}, Params(nil, Name("second"))),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:b" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestProvideParamsCanSkipFirstArgAndTagSecond(t *testing.T) {
	type out struct {
		value string
	}

	var got *out
	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Supply(&skipArgB{val: "b"}, Name("second")),
			Provide(func(a *skipArgA, b *skipArgB) *out {
				return &out{value: a.val + ":" + b.val}
			}, Params(nil, Name("second"))),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "a:b" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestInvokeParamsCanSkipFirstArgAndTagSecondWithSkipOption(t *testing.T) {
	var got string
	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Supply(&skipArgB{val: "b"}, Name("second")),
			Invoke(func(a *skipArgA, b *skipArgB) {
				got = a.val + ":" + b.val
			}, Params(Skip(), Name("second"))),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:b" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestProvideParamsCanSkipFirstArgAndTagSecondWithSkipOption(t *testing.T) {
	type out2 struct {
		value string
	}

	var got *out2
	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Supply(&skipArgB{val: "b"}, Name("second")),
			Provide(func(a *skipArgA, b *skipArgB) *out2 {
				return &out2{value: a.val + ":" + b.val}
			}, Params(Skip(), Name("second"))),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "a:b" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestInvokeParamsSkipAndOptionalSecondArg(t *testing.T) {
	var got string
	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Invoke(func(a *skipArgA, b *skipArgB) {
				if b == nil {
					got = a.val + ":nil"
					return
				}
				got = a.val + ":" + b.val
			}, Params(Skip(), Optional())),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:nil" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestInvokeParamsNilAndEmptyStringMarksSecondOptional(t *testing.T) {
	var got string
	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Invoke(func(a *skipArgA, b *skipArgB) {
				if b == nil {
					got = a.val + ":nil"
					return
				}
				got = a.val + ":" + b.val
			}, Params(nil, "")),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:nil" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestInvokeParamsSkipAndGroupSecondArg(t *testing.T) {
	var got string
	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Supply(&skipArgB{val: "g1"}, Group("workers")),
			Supply(&skipArgB{val: "g2"}, Group("workers")),
			Invoke(func(a *skipArgA, workers []*skipArgB) {
				got = a.val + ":" + strconv.Itoa(len(workers))
			}, Params(Skip(), Group("workers"))),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got != "a:2" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestProvideParamsSkipAndGroupSecondArg(t *testing.T) {
	type out3 struct {
		value string
	}
	var got *out3

	app := fxtest.New(t,
		App(
			Supply(&skipArgA{val: "a"}),
			Supply(&skipArgB{val: "g1"}, Group("workers")),
			Supply(&skipArgB{val: "g2"}, Group("workers")),
			Provide(func(a *skipArgA, workers []*skipArgB) *out3 {
				return &out3{value: a.val + ":" + strconv.Itoa(len(workers))}
			}, Params(Skip(), Group("workers"))),
			Populate(&got),
		).Build(),
	)
	defer app.RequireStart().RequireStop()

	if got == nil || got.value != "a:2" {
		t.Fatalf("unexpected result: %#v", got)
	}
}
