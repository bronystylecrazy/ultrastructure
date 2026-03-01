package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bronystylecrazy/ultrastructure/security/session"
	"github.com/bronystylecrazy/ultrastructure/x/paseto"
)

func TestPasetoManager(t *testing.T) {
	t.Run("generate and validate token pair", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:          "test-secret-key-that-is-long-enough-for-security",
			Version:         "v2",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 24 * time.Hour,
			Issuer:          "test-issuer",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Generate token pair
		pair, err := manager.Generate("user-123")
		require.NoError(t, err)
		require.NotNil(t, pair)
		assert.NotEmpty(t, pair.AccessToken)
		assert.NotEmpty(t, pair.RefreshToken)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), pair.AccessExpiresAt, 2*time.Second)
		assert.WithinDuration(t, time.Now().Add(24*time.Hour), pair.RefreshExpiresAt, 2*time.Second)

		// Validate access token
		accessClaims, err := manager.Validate(pair.AccessToken, session.TokenTypeAccess)
		require.NoError(t, err)
		assert.Equal(t, "user-123", accessClaims.Subject)
		assert.Equal(t, session.TokenTypeAccess, accessClaims.TokenType)

		// Validate refresh token
		refreshClaims, err := manager.Validate(pair.RefreshToken, session.TokenTypeRefresh)
		require.NoError(t, err)
		assert.Equal(t, "user-123", refreshClaims.Subject)
		assert.Equal(t, session.TokenTypeRefresh, refreshClaims.TokenType)
	})

	t.Run("validate wrong token type", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		// Try to validate access token as refresh
		_, err = manager.Validate(pair.AccessToken, session.TokenTypeRefresh)
		assert.Error(t, err)

		// Try to validate refresh token as access
		_, err = manager.Validate(pair.RefreshToken, session.TokenTypeAccess)
		assert.Error(t, err)
	})

	t.Run("rotate refresh token", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Generate initial pair
		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		// Rotate refresh token
		newPair, err := manager.RotateRefresh(pair.RefreshToken)
		require.NoError(t, err)
		require.NotNil(t, newPair)
		assert.NotEqual(t, pair.AccessToken, newPair.AccessToken)
		assert.NotEqual(t, pair.RefreshToken, newPair.RefreshToken)

		// Old refresh token should be revoked
		_, err = manager.Validate(pair.RefreshToken, session.TokenTypeRefresh)
		assert.Error(t, err)
	})

	t.Run("rotate access token", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Generate initial pair
		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		// Rotate access token
		newAccess, expiresAt, err := manager.RotateAccess(pair.AccessToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newAccess)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), expiresAt, 2*time.Second)

		// Old access token should be revoked
		_, err = manager.Validate(pair.AccessToken, session.TokenTypeAccess)
		assert.Error(t, err)
	})

	t.Run("middleware authenticates valid token", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Generate token
		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		// Create test handler
		var capturedSubject string
		handler := func(c fiber.Ctx) error {
			subject, err := session.SubjectFromContext(c)
			if err != nil {
				return err
			}
			capturedSubject = subject
			return c.SendString("ok")
		}

		// Create request with valid token
		app := fiber.New()
		app.Get("/test", handler, manager.AccessMiddleware())

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "user-123", capturedSubject)
	})

	t.Run("middleware rejects invalid token", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Create test handler
		handler := func(c fiber.Ctx) error {
			return c.SendString("ok")
		}

		// Create request with invalid token
		app := fiber.New()
		app.Get("/test", handler, manager.AccessMiddleware())

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("middleware rejects missing token", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Create test handler
		handler := func(c fiber.Ctx) error {
			return c.SendString("ok")
		}

		// Create request without token
		app := fiber.New()
		app.Get("/test", handler, manager.AccessMiddleware())

		req := httptest.NewRequest("GET", "/test", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("revoke token", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Generate token
		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		// Revoke refresh token
		err = manager.Revoke(context.Background(), pair.RefreshToken)
		require.NoError(t, err)

		// Token should now be invalid
		_, err = manager.Validate(pair.RefreshToken, session.TokenTypeRefresh)
		assert.Error(t, err)
	})

	t.Run("revoke from context also revokes refresh token when present", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		app := fiber.New()
		app.Post("/logout", manager.AccessMiddleware(), func(c fiber.Ctx) error {
			if err := manager.RevokeFromContext(c); err != nil {
				return err
			}
			return c.SendStatus(fiber.StatusNoContent)
		})
		app.Get("/access-protected", manager.AccessMiddleware(), func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})
		app.Get("/refresh-protected", manager.RefreshMiddleware(), func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("POST", "/logout", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: pair.RefreshToken})

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)

		accessReq := httptest.NewRequest("GET", "/access-protected", nil)
		accessReq.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		accessResp, err := app.Test(accessReq)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, accessResp.StatusCode)

		refreshReq := httptest.NewRequest("GET", "/refresh-protected", nil)
		refreshReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: pair.RefreshToken})
		refreshResp, err := app.Test(refreshReq)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, refreshResp.StatusCode)
	})

	t.Run("custom claims", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Generate with custom claims
		pair, err := manager.Generate("user-123", session.WithAccessClaims(map[string]any{
			"role":        "admin",
			"tenant":      "acme",
			"permissions": []string{"read", "write"},
		}))
		require.NoError(t, err)

		// Validate and check claims
		claims, err := manager.Validate(pair.AccessToken, session.TokenTypeAccess)
		require.NoError(t, err)
		assert.Equal(t, "admin", claims.Values["role"])
		assert.Equal(t, "acme", claims.Values["tenant"])
	})

	t.Run("extractors work with different sources", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		var capturedSubject string
		handler := func(c fiber.Ctx) error {
			subject, err := session.SubjectFromContext(c)
			if err != nil {
				return err
			}
			capturedSubject = subject
			return c.SendString("ok")
		}

		// Test header extractor
		app := fiber.New()
		app.Get("/test", handler, manager.AccessMiddleware())

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", pair.AccessToken)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "user-123", capturedSubject)
	})
}

func TestPasetoManager_Revocation(t *testing.T) {
	t.Run("revoked token is rejected in middleware", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		manager, err := session.NewPasetoManager(cfg, pv)
		require.NoError(t, err)

		// Generate token
		pair, err := manager.Generate("user-123")
		require.NoError(t, err)

		// Revoke the token
		err = manager.Revoke(context.Background(), pair.AccessToken)
		require.NoError(t, err)

		// Middleware should reject revoked token
		handler := func(c fiber.Ctx) error {
			return c.SendString("ok")
		}

		app := fiber.New()
		app.Get("/test", handler, manager.AccessMiddleware())

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

func TestUsePaseto(t *testing.T) {
	t.Run("UsePaseto provides Manager", func(t *testing.T) {
		// This is a basic DI test - in real usage this would be integrated with the DI container
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		node := session.UsePaseto(cfg)
		assert.NotNil(t, node)
	})

	t.Run("UsePasetoWithSigner provides Manager", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		pv, err := paseto.New(cfg)
		require.NoError(t, err)

		node := session.UsePasetoWithSigner(cfg, pv)
		assert.NotNil(t, node)
	})

	t.Run("WithPasetoAccessExtractors option", func(t *testing.T) {
		cfg := paseto.Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		node := session.UsePaseto(cfg,
			session.WithPasetoAccessExtractors(session.FromAuthHeader("Bearer")),
		)
		assert.NotNil(t, node)
	})
}
