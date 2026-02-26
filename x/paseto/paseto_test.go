package paseto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_WithDefaults(t *testing.T) {
	t.Run("empty config gets defaults", func(t *testing.T) {
		cfg := Config{}
		cfg = cfg.withDefaults()

		assert.Equal(t, "v2", cfg.Version)
		assert.Equal(t, defaultAccessTokenTTL, cfg.AccessTokenTTL)
		assert.Equal(t, defaultRefreshTokenTTL, cfg.RefreshTokenTTL)
		assert.Equal(t, defaultSecret, cfg.Secret)
	})

	t.Run("preserves custom values", func(t *testing.T) {
		cfg := Config{
			Version:         "v2",
			AccessTokenTTL:  30 * time.Minute,
			RefreshTokenTTL: 24 * time.Hour,
			Secret:          "my-secret-key",
			Issuer:          "test-issuer",
		}
		cfg = cfg.withDefaults()

		assert.Equal(t, "v2", cfg.Version)
		assert.Equal(t, 30*time.Minute, cfg.AccessTokenTTL)
		assert.Equal(t, 24*time.Hour, cfg.RefreshTokenTTL)
		assert.Equal(t, "my-secret-key", cfg.Secret)
		assert.Equal(t, "test-issuer", cfg.Issuer)
	})

	t.Run("v1 version is preserved", func(t *testing.T) {
		cfg := Config{Version: "v1"}
		cfg = cfg.withDefaults()
		assert.Equal(t, "v1", cfg.Version)
	})
}

func TestPaseto_SignAndVerify(t *testing.T) {
	t.Run("v2 successful sign and verify", func(t *testing.T) {
		cfg := Config{
			Secret:          "test-secret-key-that-is-long-enough-for-security",
			Version:         "v2",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 24 * time.Hour,
			Issuer:          "test-issuer",
		}

		p, err := New(cfg)
		require.NoError(t, err)
		require.NotNil(t, p)

		// Sign a token
		claims := map[string]any{
			"sub": "user-123",
			"typ": "access",
			"role": "admin",
		}

		token, err := p.Sign(claims)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		// Verify the token
		verified, err := p.Verify(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", verified.Subject)
		assert.Equal(t, "access", verified.TokenType)
		assert.Equal(t, "test-issuer", verified.Issuer)
		assert.Equal(t, "admin", verified.Values["role"])
		assert.NotEmpty(t, verified.JTI)
	})

	t.Run("v2 with short secret derives key", func(t *testing.T) {
		cfg := Config{
			Secret: "short",
			Version: "v2",
		}

		p, err := New(cfg)
		require.NoError(t, err)
		require.NotNil(t, p)

		token, err := p.Sign(map[string]any{"sub": "user-123"})
		require.NoError(t, err)

		verified, err := p.Verify(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", verified.Subject)
	})

	t.Run("expired token returns error", func(t *testing.T) {
		cfg := Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		p, err := New(cfg)
		require.NoError(t, err)

		// Create token that's already expired
		p.now = func() time.Time {
			return time.Now().UTC().Add(-1 * time.Hour)
		}

		token, err := p.Sign(map[string]any{
			"sub": "user-123",
			"exp": time.Now().UTC().Add(-30 * time.Minute).Unix(),
		})
		require.NoError(t, err)

		// Reset time for verification
		p.now = time.Now

		_, err = p.Verify(token)
		assert.Error(t, err)
	})

	t.Run("wrong version returns error", func(t *testing.T) {
		cfg := Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		p, err := New(cfg)
		require.NoError(t, err)

		// Try to verify a v1 token (which starts with "v1")
		_, err = p.Verify("v1.some.token")
		assert.ErrorIs(t, err, ErrUnexpectedTokenVersion)
	})

	t.Run("invalid token format returns error", func(t *testing.T) {
		cfg := Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		p, err := New(cfg)
		require.NoError(t, err)

		_, err = p.Verify("invalid")
		assert.Error(t, err)

		_, err = p.Verify("")
		assert.Error(t, err)
	})

	t.Run("custom claims are preserved", func(t *testing.T) {
		cfg := Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		p, err := New(cfg)
		require.NoError(t, err)

		customClaims := map[string]any{
			"sub":    "user-123",
			"role":   "admin",
			"tenant": "acme",
			"permissions": []string{"read", "write", "delete"},
		}

		token, err := p.Sign(customClaims)
		require.NoError(t, err)

		verified, err := p.Verify(token)
		require.NoError(t, err)
		assert.Equal(t, "admin", verified.Values["role"])
		assert.Equal(t, "acme", verified.Values["tenant"])
		assert.Equal(t, "user-123", verified.Subject)
	})
}

func TestPaseto_ClaimsHandling(t *testing.T) {
	t.Run("standard claims are added automatically", func(t *testing.T) {
		cfg := Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
			Issuer:  "auto-issuer",
		}

		p, err := New(cfg)
		require.NoError(t, err)

		token, err := p.Sign(map[string]any{"sub": "user-123"})
		require.NoError(t, err)

		verified, err := p.Verify(token)
		require.NoError(t, err)

		assert.NotEmpty(t, verified.JTI)
		assert.False(t, verified.IssuedAt.IsZero())
		assert.False(t, verified.NotBefore.IsZero())
		// ExpiresAt is not automatically set by Sign(), only by session managers
		assert.Equal(t, "auto-issuer", verified.Issuer)
	})

	t.Run("custom jti is preserved", func(t *testing.T) {
		cfg := Config{
			Secret:  "test-secret-key-that-is-long-enough-for-security",
			Version: "v2",
		}

		p, err := New(cfg)
		require.NoError(t, err)

		customJTI := "custom-jti-12345"
		token, err := p.Sign(map[string]any{
			"sub": "user-123",
			"jti": customJTI,
		})
		require.NoError(t, err)

		verified, err := p.Verify(token)
		require.NoError(t, err)
		assert.Equal(t, customJTI, verified.JTI)
	})
}

func TestFromMapClaims(t *testing.T) {
	t.Run("converts all claim types correctly", func(t *testing.T) {
		now := time.Now().UTC()
		input := map[string]any{
			"sub": "user-123",
			"typ": "access",
			"jti": "jti-456",
			"exp": float64(now.Add(1 * time.Hour).Unix()),
			"iat": float64(now.Unix()),
			"nbf": float64(now.Unix()),
			"iss": "test-issuer",
			"custom": "value",
		}

		claims := fromMapClaims(input)

		assert.Equal(t, "user-123", claims.Subject)
		assert.Equal(t, "access", claims.TokenType)
		assert.Equal(t, "jti-456", claims.JTI)
		assert.Equal(t, "test-issuer", claims.Issuer)
		assert.Equal(t, "value", claims.Values["custom"])
		assert.WithinDuration(t, now.Add(1*time.Hour), claims.ExpiresAt, time.Second)
	})

	t.Run("handles string time claims", func(t *testing.T) {
		now := time.Now().UTC()
		input := map[string]any{
			"sub": "user-123",
			"exp": now.Add(1 * time.Hour).Format(time.RFC3339),
		}

		claims := fromMapClaims(input)
		assert.WithinDuration(t, now.Add(1*time.Hour), claims.ExpiresAt, time.Second)
	})

	t.Run("handles int64 time claims", func(t *testing.T) {
		now := time.Now().UTC()
		input := map[string]any{
			"sub": "user-123",
			"exp": now.Add(1 * time.Hour).Unix(),
		}

		claims := fromMapClaims(input)
		assert.WithinDuration(t, now.Add(1*time.Hour), claims.ExpiresAt, time.Second)
	})
}

func TestClaims_Value(t *testing.T) {
	claims := Claims{
		Subject: "user-123",
		Values: map[string]any{
			"role": "admin",
			"tenant": "acme",
		},
	}

	t.Run("returns value for existing key", func(t *testing.T) {
		val, ok := claims.Value("role")
		assert.True(t, ok)
		assert.Equal(t, "admin", val)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		val, ok := claims.Value("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

func TestGenerateKey(t *testing.T) {
	t.Run("generates valid 32-byte key", func(t *testing.T) {
		key, err := GenerateKey()
		require.NoError(t, err)
		assert.NotEmpty(t, key)
		// Hex encoded 32 bytes = 64 characters
		assert.Len(t, key, 64)
	})

	t.Run("generates unique keys", func(t *testing.T) {
		key1, err := GenerateKey()
		require.NoError(t, err)
		key2, err := GenerateKey()
		require.NoError(t, err)
		assert.NotEqual(t, key1, key2)
	})
}

func TestDeriveKey(t *testing.T) {
	t.Run("derives consistent key from short secret", func(t *testing.T) {
		key1 := deriveKey("short")
		key2 := deriveKey("short")
		assert.Equal(t, 32, len(key1))
		assert.Equal(t, key1, key2)
	})

	t.Run("different secrets produce different keys", func(t *testing.T) {
		key1 := deriveKey("secret1")
		key2 := deriveKey("secret2")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("long secret is truncated", func(t *testing.T) {
		longSecret := "this-is-a-very-long-secret-key-that-exceeds-32-bytes"
		key := deriveKey(longSecret)
		assert.Equal(t, 32, len(key))
	})
}
