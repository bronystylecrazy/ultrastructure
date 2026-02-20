package web

import "github.com/gofiber/fiber/v3"

func NewModuleRouter(app *fiber.App, registries *RegistryContainer) Router {
	return NewRouterWithRegistry(app, registries.Metadata)
}
