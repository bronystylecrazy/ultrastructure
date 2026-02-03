package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
)

type AppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

func main() {
	err := di.App(
		di.ConfigFile("di/examples/config_watch/config.toml", di.ConfigType("toml")),
		di.Config[AppConfig]("app", di.ConfigWatch()),
		di.Invoke(func(cfg AppConfig) {
			log.Println("config", cfg.Name, cfg.Port)
		}),
	).Run()
	if err != nil {
		log.Fatal(err)
	}
}
