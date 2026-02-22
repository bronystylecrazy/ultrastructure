package jws

import (
	"strings"
	"time"
)

const (
	jwtAlgHS256 = "HS256"
	jwtAlgEdDSA = "EdDSA"

	defaultSecret           = "change-me"
	defaultSigningAlgorithm = jwtAlgHS256
	defaultAccessTokenTTL   = 15 * time.Minute
	defaultRefreshTokenTTL  = 720 * time.Hour
)

type Config struct {
	Algorithm       string        `mapstructure:"algorithm"`
	Secret          string        `mapstructure:"secret"`
	PrivateKey      string        `mapstructure:"private_key"`
	PublicKey       string        `mapstructure:"public_key"`
	PrivateKeyFile  string        `mapstructure:"private_key_file"`
	PublicKeyFile   string        `mapstructure:"public_key_file"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
}

func (c Config) withDefaults() Config {
	c.Algorithm = normalizeAlgorithm(c.Algorithm)
	if c.Algorithm == "" {
		c.Algorithm = defaultSigningAlgorithm
	}
	if c.Algorithm == jwtAlgHS256 && strings.TrimSpace(c.Secret) == "" {
		c.Secret = defaultSecret
	}
	if c.AccessTokenTTL <= 0 {
		c.AccessTokenTTL = defaultAccessTokenTTL
	}
	if c.RefreshTokenTTL <= 0 {
		c.RefreshTokenTTL = defaultRefreshTokenTTL
	}
	return c
}

func normalizeAlgorithm(v string) string {
	switch {
	case strings.EqualFold(strings.TrimSpace(v), jwtAlgHS256):
		return jwtAlgHS256
	case strings.EqualFold(strings.TrimSpace(v), jwtAlgEdDSA):
		return jwtAlgEdDSA
	default:
		return strings.ToUpper(strings.TrimSpace(v))
	}
}
