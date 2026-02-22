package authz

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func TestScopeRouteOptionStoresSecurityRequirements(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	app := fiber.New()
	r := web.NewRouterWithRegistry(app, registry)

	r.Get("/orders", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }).
		With(
			Scope(" orders:read "),
			Scopes("orders:write", "orders:read"),
		)

	meta := registry.GetRoute("GET", "/orders")
	if meta == nil {
		t.Fatal("expected route metadata")
	}
	if len(meta.Security) != 4 {
		t.Fatalf("security count: got=%d want=%d", len(meta.Security), 4)
	}

	got := map[string]struct{}{}
	for _, req := range meta.Security {
		key := req.Scheme + ":" + join(req.Scopes)
		got[key] = struct{}{}
	}
	want := []string{
		"BearerAuth:orders:read",
		"ApiKeyAuth:orders:read",
		"BearerAuth:orders:read,orders:write",
		"ApiKeyAuth:orders:read,orders:write",
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Fatalf("missing requirement: %s got=%v", k, got)
		}
	}
}
