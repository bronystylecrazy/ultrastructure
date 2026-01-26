package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us/di"
	"go.uber.org/fx"
)

type Config struct {
	Name string
}

type Store struct{}

type Service struct{}

func main() {
	fx.New(
		di.App(
			di.Diagnostics(),
			di.Invoke(func(cfg Config, st *Store, svc *Service) {
				log.Println("deps", cfg, st, svc)
			}),
		).Build(),
	).Run()
}
