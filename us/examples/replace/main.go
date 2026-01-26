package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us"
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
		us.Provide(
			func() *realReader { return &realReader{} },
			us.AsType[Reader](),
		),
		us.Replace(
			&mockReader{},
			us.AsType[Reader](),
		),
		us.Invoke(func(r Reader) {
			log.Println(r.Read())
		}),
	).Run()
}
