package sample

import (
	"strconv"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type CreateBulkRequest struct {
	Areas []string `json:"areas"`
}

type Handler struct{}

func (h *Handler) Handle(r web.Router) {
	r.Post("/templates/:template_id/bulk", h.CreateBulk)
}

func (h *Handler) CreateBulk(c fiber.Ctx) error {
	templateIDStr := c.Params("template_id")
	templateID, err := strconv.ParseInt(templateIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(web.Error{})
	}

	req := new(CreateBulkRequest)
	if err := c.Bind().Body(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(web.Error{})
	}

	if templateID == 0 {
		return c.Status(fiber.StatusConflict).JSON(web.Error{})
	}

	return c.Status(fiber.StatusCreated).JSON(web.Response{
		Data: req.Areas,
	})
}
