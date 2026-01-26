package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us/di"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		di.App(
			di.Diagnostics(),
			di.Invoke(func(msg string) {
				log.Println("message", msg)
			}),
		).Build(),
	).Run()
}
