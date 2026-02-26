package paseto

import (
	"time"
)

const (
	defaultSecret           = "change-me"
	defaultAccessTokenTTL   = 15 * time.Minute
	defaultRefreshTokenTTL  = 720 * time.Hour
)

// Config holds configuration for PASETO tokens.
type Config struct {
	// Secret is the symmetric key for V1/V2 tokens.
	// For V2, this should be 32 bytes for ChaCha20-Poly1305.
	Secret string `mapstructure:"secret"`

	// Version specifies the PASETO version ("v1" for HMAC-SHA384, "v2" for ChaCha20-Poly1305).
	// Default is "v2" for better security.
	Version string `mapstructure:"version"`

	// AccessTokenTTL is the time-to-live for access tokens.
	AccessTokenTTL time.Duration `mapstructure:"access_token_ttl"`

	// RefreshTokenTTL is the time-to-live for refresh tokens.
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`

	// Issuer is the issuer claim for tokens.
	Issuer string `mapstructure:"issuer"`
}

// withDefaults returns a copy of the config with default values applied.
func (c Config) withDefaults() Config {
	if c.Version == "" {
		c.Version = "v2"
	}
	if c.AccessTokenTTL <= 0 {
		c.AccessTokenTTL = defaultAccessTokenTTL
	}
	if c.RefreshTokenTTL <= 0 {
		c.RefreshTokenTTL = defaultRefreshTokenTTL
	}
	if c.Secret == "" {
		c.Secret = defaultSecret
	}
	return c
}
