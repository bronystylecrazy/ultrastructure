package web

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRouterWrapper_AllRegistersMultipleMethods(t *testing.T) {
	app := fiber.New()
	router := NewRouter(app)

	router.All("/all", func(c fiber.Ctx) error { return c.SendStatus(204) })

	routes := app.GetRoutes(false)
	methods := map[string]bool{}
	for _, route := range routes {
		if route.Path == "/all" {
			methods[route.Method] = true
		}
	}

	for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"} {
		if !methods[method] {
			t.Fatalf("expected %s route for /all to be registered", method)
		}
	}
}
