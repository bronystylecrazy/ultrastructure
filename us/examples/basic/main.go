package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

type test struct {
}

func NewTest() *test {
	return &test{}
}

func (t *test) Read() string {
	return "test"
}

func (t *test) Handle(r fiber.Router) {
	log.Println("test handle")
}

func main() {
	fx.New(
		us.Provide(
			NewTest,
			us.AsReaderGroup(),
			us.AsHandlerGroup(),
		),
		us.Provide(
			fx.Annotate(NewTest, us.AsReaderAnn()...),
		),
		us.Invoke(
			fx.Annotate(
				func(readers []us.Reader) {
					log.Println("annotate readers", len(readers))
				},
				fx.ParamTags(us.InReadersTag()),
			),
		),
		us.Provide(
			NewTest,
			us.AsHandlerGroup("variadic-handlers"),
		),
		us.Supply(
			&test{},
			us.AsReaderGroup("extra-readers"),
			us.AsHandlerGroup("extra-handlers"),
		),
		us.Decorate(
			func(t *test) *test {
				log.Println("decorated test")
				return t
			},
		),
		us.Invoke(func(readers []us.Reader, handlers []us.Handler) {
			log.Println("readers", len(readers))
			log.Println("handlers", len(handlers))
		}, us.InReaders(), us.InHandlers()),
		us.Invoke(func(readers []us.Reader, handlers []us.Handler) {
			log.Println("extra-readers", len(readers))
			log.Println("extra-handlers", len(handlers))
		}, us.InReaders("extra-readers"), us.InHandlers("extra-handlers")),
		us.Invoke(func(handlers ...us.Handler) {
			log.Println("variadic-handlers", len(handlers))
		}, us.InHandlers("variadic-handlers")),
	).Run()
}
