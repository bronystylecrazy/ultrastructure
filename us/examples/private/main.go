package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us"
	"go.uber.org/fx"
)

type secret struct {
	value string
}

func main() {
	fx.New(
		us.Module(
			"test",
			us.Provide(
				func() *secret { return &secret{value: "hidden"} },
				us.Private(),
			),
			us.Invoke(func(s *secret) {
				log.Println("inside module:", s.value)
			}),
		),
		us.Invoke(func(s *secret) {
			log.Println("outside module:", s.value)
		}),
	).Run()
}
