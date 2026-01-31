package main

import (
	"fmt"
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/zap"
)

func main() {
	plan, err := di.Plan(
		di.Module("core",
			di.Provide(zap.NewProduction, di.Name("prod")),
			di.Provide(zap.NewExample, di.Name("dev")),
			di.Replace(zap.NewNop),
			di.Invoke(func(l *zap.Logger) {}, di.Name("dev")),
		),
		di.Replace(zap.NewNop),
		di.Invoke(func(l *zap.Logger) {}, di.Name("prod")),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(plan)
}
