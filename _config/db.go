package config

type DbConfig struct {
	Dsn     string `mapstructure:"dsn" yaml:"dsn"`
	Migrate bool   `mapstructure:"migrate" yaml:"migrate"`
}
