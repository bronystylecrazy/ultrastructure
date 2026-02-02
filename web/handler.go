package web

import (
	"sort"

	"github.com/gofiber/fiber/v3"
)

type Handler interface {
	Handle(r fiber.Router)
}

func SetupHandlers(app *fiber.App, handlers ...Handler) {
	ordered := append([]Handler(nil), handlers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return handlerPriority(ordered[i]) < handlerPriority(ordered[j])
	})
	for _, handler := range ordered {
		handler.Handle(app)
	}
}
