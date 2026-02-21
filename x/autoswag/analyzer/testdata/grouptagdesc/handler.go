package grouptagdesc

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func Use() any { return NewHandler() }

func (h *Handler) Handle(r web.Router) {
	g := r.Group("/api/v1/px").Tags("PeopleExperience") // this is example for Sirawit
	g.Get("/hello", h.Hello)
}

func (h *Handler) Hello(c fiber.Ctx) error {
	return c.SendString("hello")
}
