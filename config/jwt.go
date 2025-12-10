package config

type JwtConfig struct {
	Secret          string `mapstructure:"secret" yaml:"secret"`
	AccessTokenTTL  string `mapstructure:"access_token_ttl" yaml:"access_token_ttl"`
	RefreshTokenTTL string `mapstructure:"refresh_token_ttl" yaml:"refresh_token_ttl"`
}
