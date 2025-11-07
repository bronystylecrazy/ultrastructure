package config

type AppConfig struct {
	Name        string `mapstructure:"name" yaml:"name"`
	Description string `mapstructure:"description" yaml:"description"`
	Port        string `mapstructure:"port" yaml:"port"`
	Debug       bool   `mapstructure:"debug" yaml:"debug"`
	LogLevel    string `mapstructure:"log_level" yaml:"log_level"`
}
