package web

import "github.com/gofiber/fiber/v3"

type Handler interface {
	Handle(r fiber.Router)
}

func SetupHandlers(app *fiber.App, handlers ...Handler) {
	for _, handler := range handlers {
		handler.Handle(app)
	}
}
