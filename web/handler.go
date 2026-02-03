package web

import (
	"sort"

	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type Handler interface {
	Handle(r fiber.Router)
}

func SetupHandlers(attached otel.Attached, app *fiber.App, handlers ...Handler) {

	ordered := append([]Handler(nil), handlers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return resolvePriority(ordered[i]) < resolvePriority(ordered[j])
	})

	for _, handler := range ordered {
		handler.Handle(app)
	}

	attached.Logger.Info("auto setup handlers", zap.Int("count", len(handlers)))
}
