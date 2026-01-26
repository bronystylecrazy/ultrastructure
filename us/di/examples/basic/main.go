package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us/di"
	"go.uber.org/fx"
)

type Handler interface {
	Handle()
}

type handlerImpl struct {
	name string
}

func (h *handlerImpl) Handle() {
	log.Println("handle", h.name)
}

func NewPrimary() *handlerImpl {
	return &handlerImpl{name: "primary"}
}

func NewGrouped() *handlerImpl {
	return &handlerImpl{name: "grouped"}
}

func main() {
	fx.New(
		di.App(
			di.Provide(
				NewPrimary,
				di.Self(),
				di.Both(
					di.As[Handler](), di.Name("primary"),
					di.As[Handler](), di.Group("handlers"),
				),
			),
			di.Provide(
				NewGrouped,
				di.As[Handler](),
				di.Group("handlers"),
			),
			di.Invoke(func(hi *handlerImpl) {
				log.Println("Self call....")
				hi.Handle()
			}),
			di.Invoke(
				func(h Handler) {
					h.Handle()
				},
				di.Name("primary"),
			),
			di.Invoke(
				func(hs ...Handler) {
					for _, h := range hs {
						h.Handle()
					}
				},
				di.Group("handlers"),
			),
		).Build(),
	).Run()
}
