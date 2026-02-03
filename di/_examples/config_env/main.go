package main

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

type Config struct {
	Service ServiceConfig `mapstructure:"service"`
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
				logger.Info("service", zap.String("service.name", cfg.Service.Name), zap.Int("service.port", cfg.Service.Port))
			}),
		).Build(),
	).Run()
}
