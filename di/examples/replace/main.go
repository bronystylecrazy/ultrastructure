package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type Reader interface {
	Read() string
}

type realReader struct{}

func (r *realReader) Read() string { return "real" }

type mockReader struct{}

func (r *mockReader) Read() string { return "mock" }

func main() {
	fx.New(
		di.App(
			di.Provide(
				func() *realReader { return &realReader{} },
				di.As[Reader](),
			),
			di.Replace(
				&mockReader{},
				di.As[Reader](),
			),
			di.Invoke(func(r Reader) {
				log.Println(r.Read())
			}),
		).Build(),
	).Run()
}
