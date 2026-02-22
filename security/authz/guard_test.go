package authz_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	authz "github.com/bronystylecrazy/ultrastructure/security/authz"
	"github.com/bronystylecrazy/ultrastructure/security/internal/testutil"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func TestGuards(t *testing.T) {
	app := fiber.New()
	app.Get("/user", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:  authn.PrincipalUser,
			Roles: []string{"admin"},
		}))
		return c.Next()
	}, authz.RequireUserRole("admin"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/app", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:   authn.PrincipalApp,
			Scopes: []string{"read:orders"},
		}))
		return c.Next()
	}, authz.RequireAppScope("read:orders"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/any-scope", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:   authn.PrincipalUser,
			Scopes: []string{"read:orders", "write:orders"},
		}))
		return c.Next()
	}, authz.RequireAnyScope("delete:orders", "write:orders"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/all-scope", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:   authn.PrincipalUser,
			Scopes: []string{"read:orders", "write:orders"},
		}))
		return c.Next()
	}, authz.RequireAllScopes("read:orders", "write:orders"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	reqUser := httptest.NewRequest(http.MethodGet, "/user", nil)
	resUser, err := app.Test(reqUser)
	if err != nil {
		t.Fatalf("app.Test(user): %v", err)
	}
	if resUser.StatusCode != fiber.StatusOK {
		t.Fatalf("user status: got=%d want=%d", resUser.StatusCode, fiber.StatusOK)
	}

	reqApp := httptest.NewRequest(http.MethodGet, "/app", nil)
	resApp, err := app.Test(reqApp)
	if err != nil {
		t.Fatalf("app.Test(app): %v", err)
	}
	if resApp.StatusCode != fiber.StatusOK {
		t.Fatalf("app status: got=%d want=%d", resApp.StatusCode, fiber.StatusOK)
	}

	reqAny := httptest.NewRequest(http.MethodGet, "/any-scope", nil)
	resAny, err := app.Test(reqAny)
	if err != nil {
		t.Fatalf("app.Test(any-scope): %v", err)
	}
	if resAny.StatusCode != fiber.StatusOK {
		t.Fatalf("any-scope status: got=%d want=%d", resAny.StatusCode, fiber.StatusOK)
	}

	reqAll := httptest.NewRequest(http.MethodGet, "/all-scope", nil)
	resAll, err := app.Test(reqAll)
	if err != nil {
		t.Fatalf("app.Test(all-scope): %v", err)
	}
	if resAll.StatusCode != fiber.StatusOK {
		t.Fatalf("all-scope status: got=%d want=%d", resAll.StatusCode, fiber.StatusOK)
	}
}

func TestPolicyPreferApp(t *testing.T) {
	userM, access := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), authz.ResolvePolicy(authz.PolicyPreferApp), func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil || p.Type != authn.PrincipalApp {
			return c.Status(fiber.StatusInternalServerError).SendString("expected app principal")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("X-API-Key", raw)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestPolicyDenyIfMultiple(t *testing.T) {
	userM, access := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), authz.ResolvePolicy(authz.PolicyDenyIfMultiple), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("X-API-Key", raw)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusForbidden)
	}
}

func TestResolvePolicyAutoEnforcesRouteScopes(t *testing.T) {
	userM, access := testutil.NewUserManager(t)

	app := fiber.New()
	container := web.NewRegistryContainer()
	r := web.NewRouterWithRegistry(app, container.Metadata)

	r.Get(
		"/p",
		authn.Any(authn.UserTokenAuthenticator(userM)),
		authz.ResolvePolicy(authz.PolicyPreferUser, authz.WithScopeRegistry(container.Metadata)),
		func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) },
	).Scopes("BearerAuth", "not:granted")
	if meta := container.Metadata.GetRoute("GET", "/p"); meta == nil || len(meta.Security) == 0 {
		t.Fatalf("expected route metadata security for GET /p, got=%v", meta)
	} else if len(meta.Security[0].Scopes) == 0 {
		t.Fatalf("expected route metadata scopes for GET /p, got=%v", meta.Security)
	}

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusForbidden)
	}
}

func TestRequireAnyScope_SuperAdminBypass(t *testing.T) {
	app := fiber.New()
	app.Get("/p", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:  authn.PrincipalUser,
			Roles: []string{authz.SuperAdminRole},
		}))
		return c.Next()
	}, authz.RequireAnyScope("missing:scope"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestRequireAllScopes_SuperAdminBypass(t *testing.T) {
	app := fiber.New()
	app.Get("/p", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:  authn.PrincipalUser,
			Roles: []string{authz.SuperAdminRole},
		}))
		return c.Next()
	}, authz.RequireAllScopes("missing:read", "missing:write"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestRequireAnyScope_CustomSuperAdminRoleBypass(t *testing.T) {
	previous := authz.SuperAdminRoles()
	authz.SetSuperAdminRoles("owner")
	t.Cleanup(func() { authz.SetSuperAdminRoles(previous...) })

	app := fiber.New()
	app.Get("/p", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:  authn.PrincipalUser,
			Roles: []string{"owner"},
		}))
		return c.Next()
	}, authz.RequireAnyScope("missing:scope"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestRequireAnyScope_SuperAdminRoleOverrideDisablesDefault(t *testing.T) {
	previous := authz.SuperAdminRoles()
	authz.SetSuperAdminRoles("owner")
	t.Cleanup(func() { authz.SetSuperAdminRoles(previous...) })

	app := fiber.New()
	app.Get("/p", func(c fiber.Ctx) error {
		c.SetContext(authn.WithPrincipal(c.Context(), &authn.Principal{
			Type:  authn.PrincipalUser,
			Roles: []string{authz.SuperAdminRole},
		}))
		return c.Next()
	}, authz.RequireAnyScope("missing:scope"), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusForbidden)
	}
}
