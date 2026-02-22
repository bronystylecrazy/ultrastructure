package dm

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

type Message struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func (h *Handler) Handle(r web.Router) {
	g := r.Group("/api/v1")

	g.Get("/dm", func(c fiber.Ctx) error {

		/*This is example command for c.SendString("Hello, World!")*/
		return c.SendString("Hello, World!")
	})

	r.Get("/dm/:id", func(c fiber.Ctx) error {

		if c.Query("detail") == "true" {
			return c.JSON(struct {
				ID   string `json:"id__id"`
				Text string `json:"text__textss"`
			}{
				ID:   c.Params("id"),
				Text: "Hello, World!",
			})
		}

		// This is example command, ei ei, for example sdjfksdjf hahaha
		return SendResponse(c)
	})

	g.Get("/gm/:id", h.GetOne)
}

func MyQuery(c fiber.Ctx) any {
	return struct {
		ID   string `json:"id__id"`
		Text string `json:"text__textss___eie"`
	}{
		ID:   c.Params("id"),
		Text: "Hello, World!",
	}
}

func SendResponse(c fiber.Ctx) error {
	return c.JSON(struct {
		ID   string `json:"id__id"`
		Text string `json:"text__textss___eie"`
	}{
		ID:   c.Params("id"),
		Text: "Hello, World!",
	})
}
