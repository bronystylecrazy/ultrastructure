package authn_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	"github.com/bronystylecrazy/ultrastructure/security/internal/testutil"
	"github.com/bronystylecrazy/ultrastructure/security/jws"
	"github.com/bronystylecrazy/ultrastructure/security/session"
	"github.com/gofiber/fiber/v3"
)

func TestEitherUserJWT(t *testing.T) {
	userM, access := testutil.NewUserManager(t)
	appM, _ := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil || p.Type != authn.PrincipalUser || p.Subject != "user-1" {
			return c.Status(fiber.StatusInternalServerError).SendString("bad principal")
		}
		if !slices.Contains(p.Scopes, "read:orders") {
			return c.Status(fiber.StatusInternalServerError).SendString("missing user scope")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestEitherAPIKey(t *testing.T) {
	userM, _ := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil || p.Type != authn.PrincipalApp || p.AppID != "app-1" {
			return c.Status(fiber.StatusInternalServerError).SendString("bad principal")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "ApiKey "+raw)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestEitherUnauthorized(t *testing.T) {
	userM, _ := testutil.NewUserManager(t)
	appM, _ := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusUnauthorized)
	}
}

func TestEitherUserJWTCookieAuthorized(t *testing.T) {
	userM, access := testutil.NewUserManager(t)
	appM, _ := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: access})
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestEitherUserJWTHeaderAuthorized(t *testing.T) {
	userM, access := testutil.NewUserManager(t)
	appM, _ := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("X-Access-Token", access)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestEitherAPIKeyCookieAuthorized(t *testing.T) {
	userM, _ := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.AddCookie(&http.Cookie{Name: "api_key", Value: raw})
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestEitherAPIKeyQueryAuthorized(t *testing.T) {
	userM, _ := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p?api_key="+raw, nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestEitherPrefersBearerWhenBothPresent(t *testing.T) {
	userM, access := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		all, ok := authn.PrincipalsFromContext(c.Context())
		if !ok || len(all) != 2 {
			return c.Status(fiber.StatusInternalServerError).SendString("expected both principals")
		}
		if p.Type != authn.PrincipalUser {
			return c.Status(fiber.StatusInternalServerError).SendString("expected user precedence")
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

func TestAnyFailFastWhenMatchedAuthenticatorErrors(t *testing.T) {
	userM, _ := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get(
		"/p",
		authn.Any(
			authn.UserTokenAuthenticator(userM),
			authn.APIKeyAuthenticator(appM),
		),
		func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	req.Header.Set("X-API-Key", raw)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusUnauthorized)
	}
}

func TestAnyBestEffortFallsBackOnAuthenticatorError(t *testing.T) {
	userM, _ := testutil.NewUserManager(t)
	appM, raw := testutil.NewAPIKeyManager(t)

	app := fiber.New()
	app.Get(
		"/p",
		authn.AnyWithMode(
			authn.ErrorModeBestEffort,
			authn.UserTokenAuthenticator(userM),
			authn.APIKeyAuthenticator(appM),
		),
		func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	req.Header.Set("X-API-Key", raw)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestEitherUserJWTRevokedUnauthorized(t *testing.T) {
	signer, err := jws.NewSigner(jws.Config{Secret: "test-secret"})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	userM, err := session.NewJWTManager(jws.Config{Secret: "test-secret"}, signer)
	if err != nil {
		t.Fatalf("NewJWTManager: %v", err)
	}
	appM, _ := testutil.NewAPIKeyManager(t)

	pair, err := userM.Generate("user-1", session.WithAccessClaims(map[string]any{
		"scope": "read:orders",
	}))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := userM.Revoke(context.Background(), pair.AccessToken); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	app := fiber.New()
	app.Get("/p", authn.Any(authn.UserTokenAuthenticator(userM), authn.APIKeyAuthenticator(appM)), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusUnauthorized)
	}
}
