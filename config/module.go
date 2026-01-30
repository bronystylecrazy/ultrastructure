package config

import "github.com/bronystylecrazy/ultrastructure/di"

func Module() di.Node {
	return di.Module(
		"us/config",
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),
	)
}
