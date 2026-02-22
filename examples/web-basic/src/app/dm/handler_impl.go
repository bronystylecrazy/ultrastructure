package dm

import "github.com/gofiber/fiber/v3"

type Response struct {
	Message string `json:"message__2"`
}

type ExampleQueryHaha struct {
	Name string `query:"name" validate:"required"` // this is example for sirawit
	Age  int    `query:"age" validate:"required"`
	Haha string `query:"haha" validate:"required"`
}

func (h *Handler) GetOne(c fiber.Ctx) error {
	query := new(ExampleQueryHaha)
	if err := c.Bind().Query(query); err != nil {
		return err
	}
	return c.JSON(Response{Message: "Hello, World!"})
}
