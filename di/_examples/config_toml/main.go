package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type AppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

type DbConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type Config struct {
	App AppConfig `mapstructure:"app"`
	Db  DbConfig  `mapstructure:"db"`
}

func main() {
	fx.New(
		di.App(
			di.Config[Config](
				"di/examples/config_toml/config.toml",
				di.ConfigType("toml"),
			),
			di.Invoke(func(cfg Config) {
				log.Println("app", cfg.App.Name, cfg.App.Port)
				log.Println("db", cfg.Db.Host, cfg.Db.Port)
			}),
		).Build(),
	).Run()
}
