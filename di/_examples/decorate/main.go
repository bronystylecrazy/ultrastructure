package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type Reader interface {
	Read() string
}

type readerImpl struct {
	name string
}

func (r *readerImpl) Read() string { return r.name }

func main() {
	fx.New(
		di.App(
			di.Provide(
				func() *readerImpl { return &readerImpl{name: "base"} },
				di.As[Reader](),
				di.Group("readers"),
			),
			di.Decorate(
				func(readers []Reader) []Reader {
					log.Println("decorating", len(readers))
					return readers
				},
				di.Group("readers"),
			),
			di.Invoke(
				func(readers []Reader) {
					for _, r := range readers {
						log.Println(r.Read())
					}
				},
				di.Group("readers"),
			),
		).Build(),
	).Run()
}
