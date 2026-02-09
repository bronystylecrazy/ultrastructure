package di

import (
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type softFailTB struct {
	t *testing.T
}

func (tb softFailTB) Logf(format string, args ...interface{}) {
	tb.t.Logf(format, args...)
}

func (tb softFailTB) Errorf(format string, args ...interface{}) {
	tb.t.Logf(format, args...)
}

func (tb softFailTB) FailNow() {}

func NewFxtestAppAllowErr(t *testing.T, opts ...fx.Option) *fxtest.App {
	t.Helper()
	return fxtest.New(softFailTB{t: t}, opts...)
}
