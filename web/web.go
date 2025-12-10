package web

import (
	"github.com/bronystylecrazy/flexinfra/lifecycle"
	"github.com/bronystylecrazy/flexinfra/logging"
	"github.com/gofiber/fiber/v2"
)

type Handler interface {
	Handle(App)
}

type Authorizer interface {
	Authorize() fiber.Handler
}

type App interface {
	lifecycle.StartStoper
	logging.Logger
	fiber.Router
}
