package main

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/zap"
)

type Logger struct {
	Log *zap.Logger
}

func NewLogger() *Logger {
	return &Logger{}
}

func (base *Logger) SetLogger(l *zap.Logger) {
	base.Log = l
}

type handler struct {
	Logger
}

func NewHandler() *handler {
	return &handler{}
}

func (h *handler) Handle() {
	h.Log.Info("Handling request")
}

type Handler interface {
	Handle()
}

type Loggerer interface {
	SetLogger(*zap.Logger)
}

func main() {
	di.App(
		di.AutoGroup[Loggerer]("loggers"),
		di.AutoGroup[Handler]("handlers"),
		di.Provide(NewHandler, di.Name("h1")),
		di.Provide(NewHandler, di.Name("h2")),
		di.Provide(NewHandler, di.Name("h3")),
		di.Provide(zap.NewDevelopment),
		di.Invoke(func(log *zap.Logger, logSetters ...Loggerer) {
			for _, logSetter := range logSetters {
				logSetter.SetLogger(log)
			}
		}, di.Optional(), di.Group("loggers")),
		di.Invoke(func(handlers ...Handler) {
			for _, handler := range handlers {
				handler.Handle()
			}
		}, di.Group("handlers")),
	).Run()
}
