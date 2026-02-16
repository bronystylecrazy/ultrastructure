package main

import (
	"embed"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/bronystylecrazy/ultrastructure/web"
)

//go:embed all:web/dist
var assets embed.FS

//go:embed all:migrations
var migrations embed.FS

func main() {
	us.New(
		web.UseSpa(web.WithSpaAssets(&assets)),
		database.UseMigrations(&migrations),
		cmd.UseBasicCommands(),
		cmd.Serve(
			realtime.UseAllowHook(),
			realtime.UseTCPListener(),
			realtime.Init(),
			web.Init(),
			otel.EnableMetrics(),
			database.RunMigrations(),
			di.Provide(NewSimpleHandler),
		),
	).Run()
}
