package fieldcomments

import (
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

type Query struct {
	Name string `query:"name" validate:"required"` // this is example for sirawit
	Age  int    `query:"age" validate:"required"`
}

type Body struct {
	Title string `json:"title"` // payload title
}

type Resp struct {
	Message string `json:"message"` // response message
}

func NewHandler() *Handler { return &Handler{} }

func Use() any { return NewHandler() }

func (h *Handler) Handle(r web.Router) {
	r.Post("/comments", h.Create)
}

func (h *Handler) Create(c fiber.Ctx) error {
	q := new(Query)
	if err := c.Bind().Query(q); err != nil {
		return err
	}
	b := new(Body)
	if err := c.Bind().Body(b); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(Resp{Message: "ok"})
}
