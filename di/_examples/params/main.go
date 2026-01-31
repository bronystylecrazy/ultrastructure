package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type DB struct {
	Name string
}

type Service struct {
	DB *DB
}

func NewDB(name string) *DB {
	return &DB{Name: name}
}

func NewService(db *DB) *Service {
	return &Service{DB: db}
}

func main() {
	fx.New(
		di.App(
			di.Provide(func() *DB { return NewDB("primary") }, di.Name("primary")),
			di.Provide(func() *DB { return NewDB("test") }, di.Name("test")),
			di.Provide(NewService, di.Params(di.Name("test"))),
			di.Invoke(func(svc *Service) {
				log.Println("service db", svc.DB.Name)
			}),
			di.Invoke(func(db *DB) {
				log.Println("invoke db", db.Name)
			}, di.Params(di.Name("primary"))),
		).Build(),
	).Run()
}
