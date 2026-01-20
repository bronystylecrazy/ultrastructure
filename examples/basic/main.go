package main

import (
	us "github.com/bronystylecrazy/ultrastructure"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(us.NewApp),
	).Run()
}
