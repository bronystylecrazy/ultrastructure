package us

import "go.uber.org/fx"

type Ultra struct {
	Options []fx.Option
}

func New(options ...fx.Option) *fx.App {
	return fx.New(options...)
}
