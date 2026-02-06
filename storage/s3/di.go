package s3

import "github.com/bronystylecrazy/ultrastructure/di"

func Module() di.Node {
	return di.Module(
		"us/storage/s3",
		di.Config[Config]("storage.s3"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
		di.Provide(NewAWSConfig),
		di.Provide(NewS3Client),
	)
}
