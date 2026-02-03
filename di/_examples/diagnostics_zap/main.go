package main

import (
	"errors"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		di.App(
			di.Provide(zap.NewDevelopment),
			di.Invoke(func(l *zap.Logger) {
				err := errors.New("boom")
				l.Error("failed", zap.Error(err))
			}),
		).Build(),
	).Run()
}
