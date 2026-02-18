package token

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bronystylecrazy/ultrastructure/caching/rd"
	usweb "github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func TestGenerateTokenPairAndRefresh(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	pair, err := svc.GenerateTokenPair("user-1", map[string]any{"role": "admin"})
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	accessClaims, err := svc.ValidateToken(pair.AccessToken, TokenTypeAccess)
	if err != nil {
		t.Fatalf("ValidateToken(access): %v", err)
	}
	if got := accessClaims.Subject; got != "user-1" {
		t.Fatalf("access sub mismatch: got=%v want=%q", got, "user-1")
	}
	if got, _ := accessClaims.Value("role"); got != "admin" {
		t.Fatalf("access custom claim mismatch: got=%v want=%q", got, "admin")
	}

	refreshClaims, err := svc.ValidateToken(pair.RefreshToken, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("ValidateToken(refresh): %v", err)
	}
	if got := refreshClaims.Subject; got != "user-1" {
		t.Fatalf("refresh sub mismatch: got=%v want=%q", got, "user-1")
	}

	nextPair, err := svc.RefreshTokenPair(pair.RefreshToken, map[string]any{"role": "admin"})
	if err != nil {
		t.Fatalf("RefreshTokenPair: %v", err)
	}
	if nextPair.AccessToken == pair.AccessToken {
		t.Fatal("expected rotated access token")
	}
	if nextPair.RefreshToken == pair.RefreshToken {
		t.Fatal("expected rotated refresh token")
	}
}

func TestAccessMiddlewareRejectsRefreshToken(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	app := fiber.New()
	app.Get("/protected", svc.AccessMiddleware(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	reqAccess := httptest.NewRequest(http.MethodGet, "/protected", nil)
	reqAccess.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	resAccess, err := app.Test(reqAccess)
	if err != nil {
		t.Fatalf("app.Test(access): %v", err)
	}
	if resAccess.StatusCode != fiber.StatusOK {
		t.Fatalf("access status: got=%d want=%d", resAccess.StatusCode, fiber.StatusOK)
	}

	reqRefresh := httptest.NewRequest(http.MethodGet, "/protected", nil)
	reqRefresh.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	resRefresh, err := app.Test(reqRefresh)
	if err != nil {
		t.Fatalf("app.Test(refresh): %v", err)
	}
	if resRefresh.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("refresh status: got=%d want=%d", resRefresh.StatusCode, fiber.StatusUnauthorized)
	}
}

func TestRefreshHandler(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	app := fiber.New()
	NewRefreshHandler(svc).Handle(usweb.NewRouter(app))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(refresh): %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("refresh status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}

	body := make([]byte, 2048)
	n, _ := res.Body.Read(body)
	if !strings.Contains(string(body[:n]), "access_token") {
		t.Fatalf("response body should include access_token, got=%q", string(body[:n]))
	}
}

func TestRefreshHandlerCookieDeliverer(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	app := fiber.New()
	NewRefreshHandler(svc).
		WithDeliverer(CookiePairDeliverer(CookiePairDelivererConfig{
			AccessCookieTemplate:  fiber.Cookie{HTTPOnly: true, Secure: true, Path: "/"},
			RefreshCookieTemplate: fiber.Cookie{HTTPOnly: true, Secure: true, Path: "/"},
		})).
		Handle(usweb.NewRouter(app))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(refresh-cookie): %v", err)
	}
	if res.StatusCode != fiber.StatusNoContent {
		t.Fatalf("refresh-cookie status: got=%d want=%d", res.StatusCode, fiber.StatusNoContent)
	}

	cookies := res.Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected at least 2 cookies, got=%d", len(cookies))
	}
}

func TestRefreshHandlerHeaderDeliverer(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	app := fiber.New()
	NewRefreshHandler(svc).
		WithDeliverer(HeaderPairDeliverer(HeaderPairDelivererConfig{})).
		Handle(usweb.NewRouter(app))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(refresh-header): %v", err)
	}
	if res.StatusCode != fiber.StatusNoContent {
		t.Fatalf("refresh-header status: got=%d want=%d", res.StatusCode, fiber.StatusNoContent)
	}
	if got := res.Header.Get("X-Access-Token"); got == "" {
		t.Fatal("expected X-Access-Token header")
	}
	if got := res.Header.Get("X-Refresh-Token"); got == "" {
		t.Fatal("expected X-Refresh-Token header")
	}
}

func TestRefreshHandlerResolverByClient(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	webDeliverer := CookiePairDeliverer(CookiePairDelivererConfig{
		AccessCookieTemplate:  fiber.Cookie{HTTPOnly: true, Path: "/"},
		RefreshCookieTemplate: fiber.Cookie{HTTPOnly: true, Path: "/"},
	})
	apiDeliverer := JSONPairDeliverer()

	app := fiber.New()
	NewRefreshHandler(svc).
		WithDelivererResolver(PairDelivererResolverFunc(func(c fiber.Ctx) PairDeliverer {
			if c.Get("X-Client-Type") == "web" {
				return webDeliverer
			}
			return apiDeliverer
		})).
		Handle(usweb.NewRouter(app))

	reqWeb := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	reqWeb.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	reqWeb.Header.Set("X-Client-Type", "web")
	resWeb, err := app.Test(reqWeb)
	if err != nil {
		t.Fatalf("app.Test(refresh-web): %v", err)
	}
	if resWeb.StatusCode != fiber.StatusNoContent {
		t.Fatalf("refresh-web status: got=%d want=%d", resWeb.StatusCode, fiber.StatusNoContent)
	}
	if len(resWeb.Cookies()) < 2 {
		t.Fatalf("refresh-web expected cookies, got=%d", len(resWeb.Cookies()))
	}

	reqAPI := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	reqAPI.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	reqAPI.Header.Set("X-Client-Type", "mobile")
	resAPI, err := app.Test(reqAPI)
	if err != nil {
		t.Fatalf("app.Test(refresh-api): %v", err)
	}
	if resAPI.StatusCode != fiber.StatusOK {
		t.Fatalf("refresh-api status: got=%d want=%d", resAPI.StatusCode, fiber.StatusOK)
	}
	body := make([]byte, 2048)
	n, _ := resAPI.Body.Read(body)
	if !strings.Contains(string(body[:n]), "access_token") {
		t.Fatalf("refresh-api body should include access_token, got=%q", string(body[:n]))
	}
}

func TestAccessMiddlewareWithChainExtractors(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	app := fiber.New()
	app.Get(
		"/protected",
		svc.AccessMiddleware(
			FromAuthHeader("Bearer"),
			FromCookie("access_token"),
		),
		func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: pair.AccessToken})
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(cookie-fallback): %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("cookie fallback status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}
}

func TestAccessMiddlewareGlobalDefaultsWithLocalOverride(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	svc.SetDefaultAccessExtractors(FromCookie("access_token"))

	app := fiber.New()
	app.Get("/global", svc.AccessMiddleware(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/override", svc.AccessMiddleware(FromAuthHeader("Bearer")), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	reqGlobal := httptest.NewRequest(http.MethodGet, "/global", nil)
	reqGlobal.AddCookie(&http.Cookie{Name: "access_token", Value: pair.AccessToken})
	resGlobal, err := app.Test(reqGlobal)
	if err != nil {
		t.Fatalf("app.Test(global): %v", err)
	}
	if resGlobal.StatusCode != fiber.StatusOK {
		t.Fatalf("global default status: got=%d want=%d", resGlobal.StatusCode, fiber.StatusOK)
	}

	reqOverride := httptest.NewRequest(http.MethodGet, "/override", nil)
	reqOverride.AddCookie(&http.Cookie{Name: "access_token", Value: pair.AccessToken})
	resOverride, err := app.Test(reqOverride)
	if err != nil {
		t.Fatalf("app.Test(override): %v", err)
	}
	if resOverride.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("override status: got=%d want=%d", resOverride.StatusCode, fiber.StatusUnauthorized)
	}
}

func TestAccessMiddlewareAllExtractors(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	app := fiber.New()
	okHandler := func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }

	app.Get("/auth-header", svc.AccessMiddleware(FromAuthHeader("Bearer")), okHandler)
	app.Get("/cookie", svc.AccessMiddleware(FromCookie("access_token")), okHandler)
	app.Get("/param/:token", svc.AccessMiddleware(FromParam("token")), okHandler)
	app.Post("/form", svc.AccessMiddleware(FromForm("token")), okHandler)
	app.Get("/header", svc.AccessMiddleware(FromHeader("X-Access-Token")), okHandler)
	app.Get("/query", svc.AccessMiddleware(FromQuery("token")), okHandler)
	app.Get("/custom", svc.AccessMiddleware(FromCustom("custom", func(c fiber.Ctx) (string, error) {
		v := c.Get("X-Custom-Token")
		if v == "" {
			return "", errors.New("not found")
		}
		return v, nil
	})), okHandler)
	app.Get("/chain", svc.AccessMiddleware(Chain(
		FromHeader("X-Primary-Token"),
		FromCookie("access_token"),
	)), okHandler)

	tests := []struct {
		name string
		req  *http.Request
	}{
		{
			name: "auth header",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/auth-header", nil)
				r.Header.Set("Authorization", "Bearer "+pair.AccessToken)
				return r
			}(),
		},
		{
			name: "cookie",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/cookie", nil)
				r.AddCookie(&http.Cookie{Name: "access_token", Value: pair.AccessToken})
				return r
			}(),
		},
		{
			name: "param",
			req:  httptest.NewRequest(http.MethodGet, "/param/"+pair.AccessToken, nil),
		},
		{
			name: "form",
			req: func() *http.Request {
				v := url.Values{}
				v.Set("token", pair.AccessToken)
				r := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(v.Encode()))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			}(),
		},
		{
			name: "header",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/header", nil)
				r.Header.Set("X-Access-Token", pair.AccessToken)
				return r
			}(),
		},
		{
			name: "query",
			req:  httptest.NewRequest(http.MethodGet, "/query?token="+pair.AccessToken, nil),
		},
		{
			name: "custom",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/custom", nil)
				r.Header.Set("X-Custom-Token", pair.AccessToken)
				return r
			}(),
		},
		{
			name: "chain fallback",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/chain", nil)
				r.AddCookie(&http.Cookie{Name: "access_token", Value: pair.AccessToken})
				return r
			}(),
		},
	}

	for _, tt := range tests {
		res, err := app.Test(tt.req)
		if err != nil {
			t.Fatalf("%s app.Test: %v", tt.name, err)
		}
		if res.StatusCode != fiber.StatusOK {
			t.Fatalf("%s status: got=%d want=%d", tt.name, res.StatusCode, fiber.StatusOK)
		}
	}
}

func TestRevokedTokenIsRejected(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	redisClient, err := rd.NewClient(rd.Config{InMemory: true})
	if err != nil {
		t.Fatalf("rd.NewClient: %v", err)
	}
	svc.SetRevocationStore(NewRedisRevocationStore(redisClient, "test:revoked:"))

	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if err := svc.RevokeToken(context.Background(), pair.AccessToken); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	app := fiber.New()
	app.Get("/protected", svc.AccessMiddleware(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(revoked): %v", err)
	}
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("revoked status: got=%d want=%d", res.StatusCode, fiber.StatusUnauthorized)
	}
}

func TestRevokeTokenWithoutStore(t *testing.T) {
	svc, err := NewService(Config{
		Secret:          "test-secret",
		AccessTokenTTL:  10 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	pair, err := svc.GenerateTokenPair("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	err = svc.RevokeToken(context.Background(), pair.AccessToken)
	if !errors.Is(err, ErrRevocationStoreNotConfigured) {
		t.Fatalf("RevokeToken without store: got=%v want=%v", err, ErrRevocationStoreNotConfigured)
	}
}

func TestRedisRevocationNamespaceIsolation(t *testing.T) {
	redisClient, err := rd.NewClient(rd.Config{InMemory: true})
	if err != nil {
		t.Fatalf("rd.NewClient: %v", err)
	}

	storeA := NewRedisRevocationStoreWithNamespace(redisClient, "test:revoked:", "app-a")
	storeB := NewRedisRevocationStoreWithNamespace(redisClient, "test:revoked:", "app-b")

	expiresAt := time.Now().Add(10 * time.Minute)
	if err := storeA.Revoke(context.Background(), "same-jti", expiresAt); err != nil {
		t.Fatalf("storeA.Revoke: %v", err)
	}

	revokedA, err := storeA.IsRevoked(context.Background(), "same-jti")
	if err != nil {
		t.Fatalf("storeA.IsRevoked: %v", err)
	}
	if !revokedA {
		t.Fatal("expected revoked in namespace app-a")
	}

	revokedB, err := storeB.IsRevoked(context.Background(), "same-jti")
	if err != nil {
		t.Fatalf("storeB.IsRevoked: %v", err)
	}
	if revokedB {
		t.Fatal("expected not revoked in namespace app-b")
	}
}

func TestRedisRevocationDefaultNamespace(t *testing.T) {
	redisClient, err := rd.NewClient(rd.Config{InMemory: true})
	if err != nil {
		t.Fatalf("rd.NewClient: %v", err)
	}

	storeDefault := NewRedisRevocationStore(redisClient, "test:revoked:")
	storeCustom := NewRedisRevocationStoreWithNamespace(redisClient, "test:revoked:", "app-x")

	expiresAt := time.Now().Add(10 * time.Minute)
	if err := storeDefault.Revoke(context.Background(), "same-jti", expiresAt); err != nil {
		t.Fatalf("storeDefault.Revoke: %v", err)
	}

	revokedDefault, err := storeDefault.IsRevoked(context.Background(), "same-jti")
	if err != nil {
		t.Fatalf("storeDefault.IsRevoked: %v", err)
	}
	if !revokedDefault {
		t.Fatal("expected revoked in default namespace")
	}

	revokedCustom, err := storeCustom.IsRevoked(context.Background(), "same-jti")
	if err != nil {
		t.Fatalf("storeCustom.IsRevoked: %v", err)
	}
	if revokedCustom {
		t.Fatal("expected not revoked in custom namespace")
	}
}
