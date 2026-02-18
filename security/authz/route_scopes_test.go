package authz_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	authz "github.com/bronystylecrazy/ultrastructure/security/authz"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func TestRequireRouteScopes_UserAllowed(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	app := fiber.New()
	r := web.NewRouterWithRegistry(app, registry)

	r.Get("/orders", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:   authn.PrincipalUser,
			Scopes: []string{"orders:read"},
		}))
		return c.Next()
	}, authz.RequireRouteScopes(authz.WithScopeRegistry(registry)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}).Scopes("BearerAuth", "orders:read")
	if meta := registry.GetRoute("GET", "/orders"); meta == nil || len(meta.Security) == 0 {
		t.Fatalf("expected route metadata security for GET /orders, got=%v", meta)
	} else if len(meta.Security[0].Scopes) == 0 {
		t.Fatalf("expected route metadata scopes for GET /orders, got=%v", meta.Security)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestRequireRouteScopes_UserForbiddenOnMissingScope(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	app := fiber.New()
	r := web.NewRouterWithRegistry(app, registry)

	r.Get("/orders", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:   authn.PrincipalUser,
			Scopes: []string{"orders:write"},
		}))
		return c.Next()
	}, authz.RequireRouteScopes(authz.WithScopeRegistry(registry)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}).Scopes("BearerAuth", "orders:read")
	if meta := registry.GetRoute("GET", "/orders"); meta == nil || len(meta.Security) == 0 {
		t.Fatalf("expected route metadata security for GET /orders, got=%v", meta)
	} else if len(meta.Security[0].Scopes) == 0 {
		t.Fatalf("expected route metadata scopes for GET /orders, got=%v", meta.Security)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusForbidden)
	}
}

func TestRequireRouteScopes_AppAllowed(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	app := fiber.New()
	r := web.NewRouterWithRegistry(app, registry)

	r.Get("/orders", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:   authn.PrincipalApp,
			Scopes: []string{"orders:read"},
		}))
		return c.Next()
	}, authz.RequireRouteScopes(authz.WithScopeRegistry(registry)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}).Scopes("ApiKeyAuth", "orders:read")
	if meta := registry.GetRoute("GET", "/orders"); meta == nil || len(meta.Security) == 0 {
		t.Fatalf("expected route metadata security for GET /orders, got=%v", meta)
	} else if len(meta.Security[0].Scopes) == 0 {
		t.Fatalf("expected route metadata scopes for GET /orders, got=%v", meta.Security)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestRequireRouteScopes_ForbiddenOnSchemeMismatch(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	app := fiber.New()
	r := web.NewRouterWithRegistry(app, registry)

	r.Get("/orders", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:   authn.PrincipalApp,
			Scopes: []string{"orders:read"},
		}))
		return c.Next()
	}, authz.RequireRouteScopes(authz.WithScopeRegistry(registry)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}).Scopes("BearerAuth", "orders:read")
	if meta := registry.GetRoute("GET", "/orders"); meta == nil || len(meta.Security) == 0 {
		t.Fatalf("expected route metadata security for GET /orders, got=%v", meta)
	} else if len(meta.Security[0].Scopes) == 0 {
		t.Fatalf("expected route metadata scopes for GET /orders, got=%v", meta.Security)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusForbidden)
	}
}
