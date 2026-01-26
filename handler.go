package us

import (
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v3"
)

type Handler interface {
	Handle(r fiber.Router)
}

func AsFiberHandler(f any) {
	spew.Dump(reflect.TypeOf(f))
}

// func AsHandler(f any) any {
// 	return fx.Provide(
// 		f,
// 		fx.Annotate(),
// 	)
// }
