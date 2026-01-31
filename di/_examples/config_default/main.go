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

func main() {
	fx.New(
		di.App(
			di.Config[AppConfig](
				"app",
				di.ConfigType("toml"),
				di.ConfigPath("di/examples/config_default"),
				di.ConfigName("config"),
				di.ConfigDefault("app.name", "test"),
			),
			di.Replace(AppConfig{Name: "Sirawit", Port: 4545}),
			di.Invoke(func(cfg AppConfig) {
				log.Println("app", cfg.Name, cfg.Port)
			}),
		).Build(),
	).Run()
}
