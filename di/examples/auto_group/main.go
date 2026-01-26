package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type Handler interface {
	Handle()
}

type Reader interface {
	Read() string
}

type A struct{}

type B struct{}

type C struct{}

func (A) Handle() { log.Println("A handle") }

func (B) Handle() { log.Println("B handle") }

func (B) Read() string { return "B read" }

func (C) Read() string { return "C read" }

func NewA() *A { return &A{} }

func NewB() *B { return &B{} }

func NewC() *C { return &C{} }

func main() {
	fx.New(
		di.App(
			di.AutoGroup[Handler]("handlers"),
			di.AutoGroup[Reader]("readers-global"),
			di.Provide(NewA),
			di.Provide(NewC, di.AutoGroupIgnore()),
			di.Module("core",
				di.AutoGroup[Reader]("readers"),
				di.Provide(NewB),
			),
			di.Invoke(func(handlers []Handler, readers []Reader) {
				log.Println("handlers", len(handlers))
				for _, h := range handlers {
					h.Handle()
				}
				log.Println("readers", len(readers))
				for _, r := range readers {
					log.Println(r.Read())
				}
			}, di.Group("handlers"), di.Group("readers")),
			di.Invoke(func(readers []Reader) {
				log.Println("readers-global", len(readers))
				for _, r := range readers {
					log.Println(r.Read())
				}
			}, di.Group("readers-global")),
		).Build(),
	).Run()
}
