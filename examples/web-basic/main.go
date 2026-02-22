package main

//go:generate env GOCACHE=/tmp/go-build go run ../../x/autoswag/cmd/autoswag-analyze -dir ../.. -patterns ./examples/web-basic/... -tags autoswag_analyze -emit-hook -hook-package main -emit-hook-name AutoSwagGenerator -out ./zz_autoswag_hook.gen.go

import (
	"embed"
	"mime/multipart"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/examples/web-basic/src/app/dm"
	"github.com/bronystylecrazy/ultrastructure/examples/web-basic/src/app/px"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/bronystylecrazy/ultrastructure/x/autoswag"
	"github.com/bronystylecrazy/ultrastructure/x/spa"
	"github.com/gofiber/fiber/v3"
)

//go:embed all:web/dist
var assets embed.FS

type ExampleHandler struct {
}

func NewExampleHandler() *ExampleHandler {
	return &ExampleHandler{}
}

type ExampleResponse struct {
	Message string `json:"message"`
}

type ExampleForm struct {
	Name string `json:"name" description:"User's name" validate:"required"`

	Avatar *multipart.FileHeader `json:"avatar" description:"User's avatar"`
}

func (e *ExampleHandler) Handle(r web.Router) {
	g := r.Group("/api/v1")
	g.Get("/", func(c fiber.Ctx) error {
		payload := new(ExampleForm)
		if err := c.Bind().Body(payload); err != nil {
			return err
		}

		return c.JSON(randomStruct())
	})

	g.Get("/haha", e.GetRandomStruct)
}

type ExampleQueryHaha struct {
	Name string `query:"name" description:"User's name" validate:"required"`
	Age  int    `query:"age" description:"User's age" validate:"required"`
	Haha string `query:"haha" description:"User's haha" validate:"required"`
}

func (e *ExampleHandler) GetRandomStruct(c fiber.Ctx) error {
	query := new(ExampleQueryHaha)
	if err := c.Bind().Query(query); err != nil {
		return err
	}

	return c.JSON(randomStruct())
}

func randomStruct() struct {
	Name   string
	Age    int
	Avatar *multipart.FileHeader
} {
	return struct {
		Name   string
		Age    int
		Avatar *multipart.FileHeader
	}{
		Name: "John Doe",
		Age:  30,
	}
}

func main() {
	us.New(
		web.UseServeCommand(),

		di.Provide(NewExampleHandler),
		di.Provide(dm.NewHandler),
		di.Provide(px.NewHandler),

		cmd.Run(
			// extensions
			autoswag.Use(
				autoswag.WitHook(AutoSwagGenerator),
				autoswag.WithEmitFiles(),
			),
			spa.Use(spa.WithAssets(&assets)),
			web.Init(),
		),
	).Run()
}
