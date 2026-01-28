package main

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/log"
	"go.uber.org/zap"
)

type Handler struct {
	log *zap.Logger
}

func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{log: logger}
}

func main() {
	di.App(
		di.Diagnostics(),
		log.Module(),
		di.Provide(NewHandler, di.As[*Handler](), di.Name("h1")),
		di.Module("test",
			di.Provide(NewHandler, di.As[*Handler](), di.Name("h2")),
		),
		di.Invoke(func(h1 *Handler, h2 *Handler) {
			h1.log.Info("Hello1")
			h2.log.Info("Hello2")
		}, di.Name("h1"), di.Name("h2")),
	).Run()
}
