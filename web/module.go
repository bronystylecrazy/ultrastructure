package web

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
)

var HandlersGroupName = "us.handlers"

func Module(extends ...di.Node) di.Node {
	return di.Module(
		"us/web",
		di.Config[Config]("web"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),

		di.Config[FiberConfig]("web"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),
		di.Provide(NewFiberApp, di.As[fiber.Router](), di.AsSelf()),

		// auto discovery for handlers
		di.AutoGroup[Handler](HandlersGroupName),

		di.Provide(NewZapMiddleware),
		di.Options(di.ConvertAnys(extends)...),
		di.Invoke(SetupHandlers, di.Params(``, di.Group(HandlersGroupName))),
		di.Invoke(func(config Config, app *fiber.App) {
			go app.Listen(fmt.Sprintf("%s:%d", config.Host, config.Port))
		}),
	)
}
