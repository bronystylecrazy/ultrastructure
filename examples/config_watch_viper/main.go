package main

import (
	"context"
	"log"

	"github.com/bronystylecrazy/ultrastructure/us"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type AppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

func main() {
	v := viper.New()
	v.SetConfigFile("config.toml")
	v.SetDefault("app.name", "demo")
	v.SetDefault("app.port", 8080)

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("read config: %v", err)
	}

	restart := make(chan struct{}, 1)
	v.OnConfigChange(func(_ fsnotify.Event) {
		select {
		case restart <- struct{}{}:
		default:
		}
	})
	v.WatchConfig()

	build := func() *fx.App {
		return fx.New(
			fx.NopLogger,
			fx.Supply(v),
			fx.Invoke(func(cfg AppConfig) {
				log.Printf("app started: name=%s port=%d", cfg.Name, cfg.Port)
			}),
			fx.Provide(func(v *viper.Viper) (AppConfig, error) {
				var cfg AppConfig
				return cfg, v.UnmarshalKey("app", &cfg)
			}),
		)
	}

	watcher := us.NewAppWatcher(build, restart)
	if err := watcher.Run(context.Background()); err != nil {
		log.Fatalf("app watcher stopped: %v", err)
	}
}
