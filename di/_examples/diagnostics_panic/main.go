package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func NewThing() *int {
	panic("hllll")
}

func main() {
	fx.New(
		di.App(
			di.Diagnostics(),
			di.Provide(NewThing),
			di.Invoke(func(v *int) {
				log.Println("value", *v)
			}),
		).Build(),
	).Run()
}
