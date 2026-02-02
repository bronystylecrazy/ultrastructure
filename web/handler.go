package web

import (
	"log"
	"sort"

	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/gofiber/fiber/v3"
)

type Handler interface {
	Handle(r fiber.Router)
}

func SetupHandlers(_ otel.Attached, app *fiber.App, handlers ...Handler) {
	log.Println("attaching telemetry to handlers", len(handlers))
	ordered := append([]Handler(nil), handlers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return resolvePriority(ordered[i]) < resolvePriority(ordered[j])
	})
	for _, handler := range ordered {
		handler.Handle(app)
	}
}
