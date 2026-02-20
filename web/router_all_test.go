package web

import (
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRouterWrapper_AllRegistersMultipleMethods(t *testing.T) {
	app := fiber.New()
	router := NewRouterWithRegistry(app, NewMetadataRegistry())

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

func TestRouterWrapper_WithAppliesDefaultRouteOptions(t *testing.T) {
	registry := NewMetadataRegistry()
	app := fiber.New()
	type createUserBody struct {
		Name string `json:"name"`
	}
	router := NewRouterWithRegistry(app, registry).
		With(
			BadRequest[Error]("invalid request"),
			Body(createUserBody{}),
		)

	router.Post("/users", func(c fiber.Ctx) error { return c.SendStatus(201) })

	meta := registry.GetRoute("POST", "/users")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	if _, ok := meta.Responses[400]; !ok {
		t.Fatalf("expected default 400 response from router.With")
	}
	if meta.RequestBody == nil {
		t.Fatalf("expected default request body from router.With")
	}
	if got := meta.RequestBody.Content[ContentTypeApplicationJSON]; got != reflect.TypeOf(createUserBody{}) {
		t.Fatalf("expected default request body content type application/json, got %v", got)
	}
}

func TestRouterWrapper_WithDefaultsInheritedByGroup(t *testing.T) {
	registry := NewMetadataRegistry()
	app := fiber.New()
	router := NewRouterWithRegistry(app, registry).With(BadRequest[Error]("invalid request"))

	users := router.Group("/users")
	users.Get("/:id", func(c fiber.Ctx) error { return c.SendStatus(200) })

	meta := registry.GetRoute("GET", "/users/:id")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	if _, ok := meta.Responses[400]; !ok {
		t.Fatalf("expected inherited default 400 response on group route")
	}
}

func TestRouterWrapper_GroupTagsPersistWithoutBuilderChaining(t *testing.T) {
	registry := NewMetadataRegistry()
	app := fiber.New()

	g := NewRouterWithRegistry(app, registry).
		Group("/api/v1/px").
		Tags("PeopleExperience")

	g.Get("/", func(c fiber.Ctx) error { return c.SendStatus(200) })

	meta := registry.GetRoute("GET", "/api/v1/px")
	if meta == nil {
		t.Fatalf("expected route metadata to be auto-registered")
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "PeopleExperience" {
		t.Fatalf("expected inherited group tag PeopleExperience, got %v", meta.Tags)
	}
}

func TestMetadataRegistry_PathLookupIgnoresTrailingSlash(t *testing.T) {
	registry := NewMetadataRegistry()
	app := fiber.New()

	g := NewRouterWithRegistry(app, registry).
		Group("/api/v1/px").
		Tags("PeopleExperience")

	g.Get("/", func(c fiber.Ctx) error { return c.SendStatus(200) })

	meta := registry.GetRoute("GET", "/api/v1/px/")
	if meta == nil {
		t.Fatalf("expected metadata lookup to match trailing slash variant")
	}
	if len(meta.Tags) != 1 || meta.Tags[0] != "PeopleExperience" {
		t.Fatalf("expected PeopleExperience tag, got %v", meta.Tags)
	}
}
