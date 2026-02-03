package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
)

type ServiceConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

func main() {
	err := di.App(
		di.ConfigFile("di/examples/config_watch/config.toml", di.ConfigType("toml")),
		di.Config[ServiceConfig]("service", di.ConfigWatch()),
		di.Invoke(func(cfg ServiceConfig) {
			log.Println("config", cfg.Name, cfg.Port)
		}),
	).Run()
	if err != nil {
		log.Fatal(err)
	}
}
