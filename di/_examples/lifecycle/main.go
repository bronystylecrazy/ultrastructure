package main

import (
	"context"
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		di.App(
			di.OnStart(func(ctx context.Context) error {
				_ = ctx
				log.Println("starting")
				return nil
			}),
			di.OnStop(func(ctx context.Context) error {
				_ = ctx
				log.Println("stopping")
				return nil
			}),
			di.Invoke(func() {
				log.Println("ready")
			}),
		).Build(),
	).Run()
}
