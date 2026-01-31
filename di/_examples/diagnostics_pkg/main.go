package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/di/examples/diagnostics_pkg/helper"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		di.App(
			di.Diagnostics(),
			di.Provide(helper.NewService),
			di.Invoke(func(s *helper.Service) {
				log.Println("service", s)
			}),
		).Build(),
	).Run()
}
