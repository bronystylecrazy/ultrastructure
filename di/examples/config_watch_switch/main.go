package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us/di"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type AppConfig struct {
	Log string `mapstructure:"log"`
}

type DbConfig struct {
	Message string `mapstructure:"message"`
}

func main() {
	err := di.App(
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),
		di.ConfigFile("di/examples/config_watch_switch/config.toml", di.ConfigType("toml")),
		di.Config[AppConfig]("app", di.ConfigWatch()),
		di.Config[DbConfig]("db", di.ConfigWatch()),
		di.Switch(
			di.WhenCase(func(cfg AppConfig) bool { return cfg.Log == "prod" },
				di.Provide(zap.NewProduction),
			),
			di.WhenCase(func(cfg AppConfig, dbCfg DbConfig) bool {
				return cfg.Log == "dev"
			}, di.Provide(zap.NewDevelopment)),
			di.DefaultCase(
				di.Provide(zap.NewExample),
			),
		),
		di.Invoke(func(l *zap.Logger, cfg DbConfig) {
			l.Info(cfg.Message)
		}),
	).Run()
	if err != nil {
		log.Fatal(err)
	}
}
