package web

type Config struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}
