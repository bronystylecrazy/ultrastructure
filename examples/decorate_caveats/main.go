package main

import "github.com/bronystylecrazy/ultrastructure/di"

type Handler interface{ Handle() string }
type H struct{ id string }

func (h *H) Handle() string { return h.id }

func NewH() *H { return &H{id: "raw"} }

func main() {

	di.App(
		di.AutoGroup[Handler]("handlers"),
		di.Provide(NewH), // exports *H (and auto-grouped Handler group)
		di.Decorate(func(h *H) *H {
			h.id = "decorated"
			return h
		}),
		di.Invoke(func(handlers ...Handler) {
			for _, handler := range handlers {
				println(handler.Handle())
			}
		}, di.Params(`group:"handlers"`)),
	).Run()
}
