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

func main() {
	fx.New(
		di.App(
			di.Config[ServiceConfig](
				"service",
				di.ConfigType("toml"),
				di.ConfigPath("di/examples/config_default"),
				di.ConfigName("config"),
				di.ConfigDefault("service.name", "test"),
			),
			di.Replace(ServiceConfig{Name: "CONNECTEDTECH", Port: 4545}),
			di.Invoke(func(cfg ServiceConfig) {
				log.Println("service", cfg.Name, cfg.Port)
			}),
		).Build(),
	).Run()
}
