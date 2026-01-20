package web

type Config struct {
	Name        string `mapstructure:"name" yaml:"name"`
	Description string `mapstructure:"description" yaml:"description"`
	Host        string `mapstructure:"host" yaml:"host"`
	Port        string `mapstructure:"port" yaml:"port"`
}
