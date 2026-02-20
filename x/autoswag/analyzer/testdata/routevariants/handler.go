package routevariants

import (
	"strconv"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

type confidenceBody struct {
	Name string `json:"name"`
}

type confidenceQuery struct {
	Q string `query:"q"`
}

func (h *Handler) Handle(r web.Router) {
	g := r.Group("/v")
	_ = g.Get("/assign", h.AssignRoute)
	var _ = g.Get("/decl", h.DeclRoute)
	g.Get("/multi", h.MultiResponse)
	g.Get("/ambiguous", h.AmbiguousResponse)
	g.Get("/pathonly/:slug", h.PathOnly)
	g.Post("/confidence/:id", h.Confidence)
}

func (h *Handler) AssignRoute(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) DeclRoute(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusAccepted)
}

func (h *Handler) MultiResponse(c fiber.Ctx) error {
	if c.Query("format") == "text" {
		return c.Status(fiber.StatusOK).SendString("ok")
	}
	return c.Status(fiber.StatusOK).JSON(web.Response{})
}

func (h *Handler) AmbiguousResponse(c fiber.Ctx) error {
	if c.Query("err") == "1" {
		return c.Status(fiber.StatusOK).JSON(web.Error{})
	}
	return c.Status(fiber.StatusOK).JSON(web.Response{})
}

func (h *Handler) PathOnly(c fiber.Ctx) error {
	slug := c.Params("slug")
	_ = slug
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Confidence(c fiber.Ctx) error {
	id := c.Params("id")
	_, _ = strconv.ParseInt(id, 10, 64)

	req := new(confidenceBody)
	body := req
	if err := c.Bind().Body(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(web.Error{})
	}

	q := new(confidenceQuery)
	query := q
	_ = c.Bind().Query(query)

	return c.Status(fiber.StatusCreated).JSON(web.Response{})
}
