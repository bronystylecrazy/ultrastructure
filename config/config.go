package config

import "go.uber.org/fx"

type Config struct {
	fx.Out `yaml:"-"`

	App *AppConfig `mapstructure:"app" yaml:"app"`
	Jwt *JwtConfig `mapstructure:"jwt" yaml:"jwt"`
	Db  *DbConfig  `mapstructure:"db" yaml:"db"`
}
