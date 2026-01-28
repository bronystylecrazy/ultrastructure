package web

import (
	"github.com/gofiber/fiber/v2"
)

type Handler interface {
	Handle(r fiber.Router)
}

type Authorizer interface {
	Authorize() fiber.Handler
}
