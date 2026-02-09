package web

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
)

var HandlersGroupName = "us.handlers"

func Module(extends ...di.Node) di.Node {
	return di.Module(
		"us.web",
		// auto discovery for handlers
		di.AutoGroup[Handler](HandlersGroupName), // only available inside this module

		di.Config[Config]("web"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),

		di.Config[FiberConfig]("web"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
		di.Provide(NewFiberApp, di.AsSelf[fiber.Router]()),

		di.Options(di.ConvertAnys(extends)...),
		di.Invoke(SetupHandlers),
		di.Invoke(RegisterFiberApp),
	)
}
