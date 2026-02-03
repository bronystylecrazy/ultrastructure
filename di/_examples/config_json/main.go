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

type Config struct {
	Service ServiceConfig `mapstructure:"service"`
	Db      DbConfig      `mapstructure:"db"`
}

func main() {
	fx.New(
		di.App(
			di.Config[Config]("di/examples/config_json/config.json", di.ConfigType("json")),
			di.Invoke(func(cfg Config) {
				log.Println("service", cfg.Service.Name, cfg.Service.Port)
				log.Println("db", cfg.Db.Host, cfg.Db.Port)
			}),
		).Build(),
	).Run()
}
