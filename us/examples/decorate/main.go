package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/us"
	"go.uber.org/fx"
)

type readerImpl struct {
	name string
}

func (r *readerImpl) Read() string {
	return r.name
}

func main() {
	fx.New(
		us.Invoke(func(reader []us.Reader) {
			for _, r := range reader {
				log.Println(r.Read())
			}
		}, us.InReaders()),
	).Run()
}
