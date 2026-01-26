package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us/di"
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

func main() {
	fx.New(
		di.App(
			di.ConfigBind[AppConfig](
				"app",
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
			di.Invoke(func(app AppConfig, db DbConfig) {
				log.Println("app", app.Name, app.Port)
				log.Println("db", db.Host, db.Port)
			}),
		).Build(),
	).Run()
}
