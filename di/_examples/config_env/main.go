package main

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

type Config struct {
	App AppConfig `mapstructure:"app"`
}

func main() {
	fx.New(
		di.App(
			di.Provide(zap.NewProduction),
			di.Config[Config](
				"di/examples/config_env/config.toml",
				di.ConfigEnvOverride(),
			),
			di.Invoke(func(cfg Config, logger *zap.Logger) {
				logger.Info("app", zap.String("app.name", cfg.App.Name), zap.Int("app.port", cfg.App.Port))
			}),
		).Build(),
	).Run()
}
