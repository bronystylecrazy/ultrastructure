package token

import "time"

const (
	defaultSecret          = "change-me"
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 720 * time.Hour
)

type Config struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
}

func (c Config) withDefaults() Config {
	if c.Secret == "" {
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
