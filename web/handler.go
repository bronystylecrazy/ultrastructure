package web

import (
	"sort"

	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler interface {
	Handle(r Router)
}

type setupHandlersIn struct {
	fx.In
	Attached otel.Attached `optional:"true"`
	App      *fiber.App
	Handlers []Handler `group:"us.handlers"`
}

func SetupHandlers(in setupHandlersIn) {

	ordered := append([]Handler(nil), in.Handlers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return resolvePriority(ordered[i]) < resolvePriority(ordered[j])
	})

	// Create router wrapper for fluent API
	router := NewRouter(in.App)

	for _, handler := range ordered {
		handler.Handle(router)
	}

	logger := in.Attached.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Debug("auto setup handlers", zap.Int("count", len(in.Handlers)))
}
