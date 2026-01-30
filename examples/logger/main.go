package main

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/log"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type handler struct {
	name string
	log  *zap.Logger
}

func NewHandler(logger *zap.Logger) *handler {
	return &handler{log: logger, name: uuid.NewString()}
}

func (h *handler) Handle(r string) {
	h.log.Info("Running", zap.String("name", h.name), zap.String("mode", r))
}

type Handler interface {
	Handle(r string)
}

func main() {
	di.App(
		log.Module(),
		di.Diagnostics(),
		di.AutoGroup[Handler]("auto-handlers"),
		di.Provide(zap.NewProduction, di.Name("l2")),
		di.Module("test1",
			di.Provide(NewHandler),
			di.Decorate(func(logger *zap.Logger) *zap.Logger {
				return logger.With(zap.String("module", "test1"))
			}),
		),
		di.Module("test2",
			di.Provide(NewHandler,
				di.As[Handler](`name:"h2"`, `group:"handlers"`),
				di.Params(`name:"l2"`),
			),
			di.Decorate(func(logger *zap.Logger) *zap.Logger {
				return logger.With(zap.String("module", "test2"))
			}),
		),
		di.Invoke(func(log *zap.Logger, h1 *handler, h2 Handler, autoHandlers []Handler, handlers ...Handler) {
			h1.Handle("manual")
			h2.Handle("manual")
			log.Info("handlers", zap.Int("handlers", len(handlers)))
			for _, h := range handlers {
				h.Handle("manual")
			}
			log.Info("autoHandlers", zap.Int("autoHandlers", len(autoHandlers)))
			for _, h := range autoHandlers {
				h.Handle("auto")
			}
		}, di.Params(``, ``, `name:"h2"`, `group:"auto-handlers"`, di.Group("handlers"))),
	).Run()
}
