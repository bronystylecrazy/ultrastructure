package web

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/gofiber/fiber/v3"
)

var HandlersGroupName = "us.handlers"

func Module(extends ...di.Node) di.Node {
	nodes := []any{
		di.Config[Config]("web"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),

		di.Config[FiberConfig]("web"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),
		di.Provide(NewFiberApp, di.As[fiber.Router](), di.AsSelf()),

		// auto discovery for handlers
		di.AutoGroup[Handler](HandlersGroupName),
		di.Invoke(SetupHandlers, di.Params(``, di.Group(HandlersGroupName))),

		di.Provide(NewZapMiddleware),
	}

	if len(extends) > 0 {
		nodes = append(nodes, di.ConvertAnys(extends)...)
	}

	nodes = append(nodes, di.Invoke(func(config Config, app *fiber.App) {
		app.Listen(fmt.Sprintf("%s:%d", config.Host, config.Port))
	}))

	return di.Module("us/web", nodes...)
}
