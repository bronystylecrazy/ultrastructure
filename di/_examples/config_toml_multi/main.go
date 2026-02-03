package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type ServiceConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

type DbConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func main() {
	fx.New(
		di.App(
			di.ConfigBind[ServiceConfig](
				"service",
				di.ConfigType("toml"),
				di.ConfigPath("di/examples/config_toml"),
				di.ConfigName("config"),
			),
			di.ConfigBind[DbConfig](
				"db",
				di.ConfigType("toml"),
				di.ConfigPath("di/examples/config_toml"),
				di.ConfigName("config"),
			),
			di.Invoke(func(service ServiceConfig, db DbConfig) {
				log.Println("service", service.Name, service.Port)
				log.Println("db", db.Host, db.Port)
			}),
		).Build(),
	).Run()
}
