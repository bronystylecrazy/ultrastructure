package web

import (
	"go.uber.org/fx"
)

type Handler interface {
	Handle(Router)
}

func AsHandler(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Handler)),
		fx.ResultTags(`group:"web.handlers"`),
	)
}

func WithHandlers(f any) any {
	return fx.Annotate(f, fx.ParamTags(`group:"routes"`))
}

func SetupHandlers(handlers []Handler, app Router) {
	for _, handler := range handlers {
		handler.Handle(app)
	}
}
