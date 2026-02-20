package web

import "go.uber.org/fx"

type Handler interface {
	Handle(r Router)
}

type setupHandlersIn struct {
	fx.In
	Router   Router
	Handlers []Handler `group:"us.handlers"`
}

func SetupHandlers(in setupHandlersIn) {
	orderedHandlers := Prioritize(in.Handlers)
	for i := range orderedHandlers {
		orderedHandlers[i].Handle(in.Router)
	}
}
