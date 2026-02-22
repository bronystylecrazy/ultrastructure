package px

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(r web.Router) {
	g := r.Group("/api/v1/px").Tags("PeopleExperience") // this is example for Sirawit

	g.Get("/description", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	}).TaggedName("GetOne")

	// hello world
	g.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	}).TaggedName("GetOne")

	g.Get("/health", func(c fiber.Ctx) error {
		if c.Query("detail") == "true" {
			// this is example for Sirawit
			return c.Status(fiber.StatusCreated).JSON(web.Response{Data: "Hello World"})
		}

		// this is example for Hello World
		return c.JSON(struct {
			IHereThundie string `json:"i_here_thundie" description:"i'm here thundie"`
		}{})
	})

	g.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	g.Get("/users/:id_or_uuid/pooler", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})
}
