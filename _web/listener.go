package web

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func Listen(app *fiber.App, config Config) error {
	return app.Listen(fmt.Sprintf("%s:%s", config.Host, config.Port))
}
