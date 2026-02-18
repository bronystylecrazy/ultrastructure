package apikey

import "time"

const (
	defaultHeaderName      = "Authorization"
	defaultHeaderScheme    = "ApiKey"
	defaultHasherMode      = "argon2id"
	defaultIDLength        = 16
	defaultSecretLength    = 32
	minIDLength            = 12
	minSecretLength        = 24
	defaultKeyPrefix       = "usk"
	defaultSkewAllowance   = 30 * time.Second
	defaultSetPrincipalCtx = true
)

type Config struct {
	HeaderName       string        `mapstructure:"header_name"`
	HeaderScheme     string        `mapstructure:"header_scheme"`
	HasherMode       string        `mapstructure:"hasher_mode"`
	HMACSecret       string        `mapstructure:"hmac_secret"`
	KeyPrefix        string        `mapstructure:"key_prefix"`
	KeyIDLength      int           `mapstructure:"key_id_length"`
	SecretLength     int           `mapstructure:"secret_length"`
	SkewAllowance    time.Duration `mapstructure:"skew_allowance"`
	SetPrincipalCtx  bool          `mapstructure:"set_principal_ctx"`
	SetPrincipalBody bool          `mapstructure:"set_principal_body"`
	DetailedErrors   bool          `mapstructure:"detailed_errors"`
}

func (c Config) withDefaults() Config {
	if c.HeaderName == "" {
		c.HeaderName = defaultHeaderName
	}
	if c.HeaderScheme == "" {
		c.HeaderScheme = defaultHeaderScheme
	}
	if c.HasherMode == "" {
		c.HasherMode = defaultHasherMode
	}
	if c.KeyPrefix == "" {
		c.KeyPrefix = defaultKeyPrefix
	}
	if c.KeyIDLength <= 0 {
		c.KeyIDLength = defaultIDLength
	}
	if c.KeyIDLength < minIDLength {
		c.KeyIDLength = minIDLength
	}
	if c.SecretLength <= 0 {
		c.SecretLength = defaultSecretLength
	}
	if c.SecretLength < minSecretLength {
		c.SecretLength = minSecretLength
	}
	if c.SkewAllowance <= 0 {
		c.SkewAllowance = defaultSkewAllowance
	}
	if !c.SetPrincipalCtx && !c.SetPrincipalBody {
		c.SetPrincipalCtx = defaultSetPrincipalCtx
	}
	return c
}
