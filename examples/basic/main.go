package main

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

type H struct {
}

// func (h *H) Handle(r fiber.Router) {
// 	r.Get("/", func(c fiber.Ctx) error {
// 		return c.SendString("Hello, World!")
// 	})
// }

func (h *H) Handle(m net.Conn) {

}

type Conner interface {
	Handle(m net.Conn)
}

type Router interface {
	Handle(r fiber.Router)
}

func AsHandler(f any) {
	t := reflect.TypeOf(f)

	now := time.Now()
	spew.Dump(t.Implements(reflect.TypeOf((*Router)(nil)).Elem()))
	fmt.Println("Elapsed time:", time.Since(now))
}

type Hello string
type World string

type Config struct {
	fx.In

	Hello Hello `name:"hello"`
	World World `name:"world"`
}

func InvokeHelloWorld(hello Hello, world World, conners ...Conner) {

	fmt.Println(
		string(hello),
		string(world),
		conners,
	)
}

func NewHandler(config Config) Conner {
	return &H{}
}

type Args struct {
	// fx.In

	Hello Hello `name:"hello"`
	World World `name:"world"`
}

type Results struct {
	// fx.Out

	Hello Hello `name:"hello"`
	World World `name:"world"`
}

func Module()

func NewApp() *fx.App {
	return fx.New(
		fx.Supply(Results{
			Hello: "hello",
			World: "worldd",
		}),
		DisableTelemetry(),
		fx.Invoke(func(lc fx.Lifecycle, r Results) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					fmt.Println("STARTED", r.Hello, r.World)
					return nil
				},
				OnStop: func(context.Context) error {
					fmt.Println("STOPPED", r.Hello, r.World)
					return nil
				},
			})
		}),
	)
}

type AppOptions map[string]any

func main() {

	fx.New(
		Configuration(),
		WebModule(
			us.UseFiber(),
			us.WithTelemetry(false),
		),
	)

	fx.New(
		fx.NopLogger,
		fx.Provide(NewApp),
		fx.Invoke(func(lc fx.Lifecycle, app *fx.App) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					app.Start(context.Background())
					return nil
				},
				OnStop: func(context.Context) error {
					app.Stop(context.Background())
					return nil
				},
			})
		}),
	).Run()
	// fx.Supply(Results{
	// 	Hello: "hello",
	// 	World: "world",
	// }),
	// fx.Module("internal1",
	// 	fx.Decorate(func(r Results) Results {
	// 		r.World += "1"
	// 		return r
	// 	}),
	// 	fx.Invoke(func(p Results) {
	// 		fmt.Println(p.Hello, p.World)
	// 	}),
	// ),
	// fx.Module("internal2",
	// 	fx.Decorate(func(r Results) Results {
	// 		r.World += "2"
	// 		return r
	// 	}),
	// 	fx.Invoke(func(p Results) {
	// 		fmt.Println(p.Hello, p.World)
	// 	}),
	// ),
	// fx.Invoke(func(p Results) {
	// 	fmt.Println(p.Hello, p.World)
	// }),
	// fx.Supply(fx.Annotated{Target: Hello("Hello"), Name: "hello"}),
	// fx.Supply(fx.Annotated{Target: World("World"), Name: "world"}),
	// fx.Supply(Config{
	// 	Hello: "config-hello",
	// 	World: "config-world",
	// }),
	// fx.Module(
	// 	"test",
	// 	fx.Replace(Hello("World")),
	// 	fx.Invoke(fx.Annotate(InvokeHelloWorld, fx.ParamTags(`name:"hello"`, `name:"world"`, `group:"handlers"`))),
	// ),
	// fx.Provide(fx.Annotated{Target: NewHandler, Group: "handlers"}),
	// fx.Supply(Hello("Sirawit")),
	// fx.Supply(World("World")),
	// fx.Invoke(func(hello Hello, world World) {
	// 	fmt.Println(
	// 		string(hello),
	// 		string(world),
	// 	)
	// }),
	// )
	// h := &H{}
	// AsHandler(h)
	// us.New(
	// 	us.UseFiber(
	// 		us.WithWebsocket(),
	// 		us.WithSpa(),
	// 		us.WithLogger()
	// 	),
	// ).Start()
}
