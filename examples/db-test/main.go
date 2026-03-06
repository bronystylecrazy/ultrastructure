package main

import (
	"embed"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/bronystylecrazy/ultrastructure/web"
	xgoose "github.com/bronystylecrazy/ultrastructure/x/goose"
	"github.com/bronystylecrazy/ultrastructure/x/spa"
)

//go:embed all:web/dist
var assets embed.FS

//go:embed all:migrations
var migrations embed.FS

func main() {
	us.New(
		spa.Use(&assets),
		xgoose.Use(&migrations),
		cmd.UseBasicCommands(),
		cmd.Run(
			realtime.UseAllowHook(),
			realtime.UseTCPListener(),
			realtime.Init(),
			web.Init(),
			otel.EnableMetrics(),
			xgoose.Run(),
			di.Provide(NewSimpleHandler),
		),
	).Run()
}
