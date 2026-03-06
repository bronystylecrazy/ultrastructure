package main

import (
	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/di"
	custom "github.com/bronystylecrazy/ultrastructure/examples/autoswagger-typed/area"
	area "github.com/bronystylecrazy/ultrastructure/examples/autoswagger-typed/custom"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/bronystylecrazy/ultrastructure/x/autoswag"
)

type UsersError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UsersSwaggerCustomizer demonstrates DI-provided per-route swagger customization.
type UsersSwaggerCustomizer struct{}

func NewUsersSwaggerCustomizer() *UsersSwaggerCustomizer {
	return &UsersSwaggerCustomizer{}
}

func (c *UsersSwaggerCustomizer) CustomizeSwagger(ctx *autoswag.Context) {
	ctx.Metadata.OperationID = ctx.RouteModelPackageName() + "_" + ctx.Metadata.OperationID
}

func main() {
	us.New(
		web.Init(),
		di.Provide(custom.NewUserHandler),
		di.Provide(area.NewUserHandler),
		di.Provide(NewUsersSwaggerCustomizer),
		autoswag.Use(),
	).Run()
}
