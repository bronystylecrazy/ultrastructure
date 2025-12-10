package config

type AppConfig struct {
	Debug    bool   `mapstructure:"debug" yaml:"debug"`
	LogLevel string `mapstructure:"log_level" yaml:"log_level"`
}
