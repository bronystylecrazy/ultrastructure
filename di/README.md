# di

`di` is a declarative layer on top of Uber Fx. It lets you compose apps from
small nodes (Provide, Invoke, Module, etc.), generate a plan, and build an
`fx.Option` with replacements, conditionals, config, and diagnostics.

## Installation

```
go get github.com/bronystylecrazy/ultrastructure/di
```

## Quick start

Minimal app with a single provide + invoke.

```go
package main

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		di.App(
			di.Provide(zap.NewDevelopment),
			di.Invoke(func(l *zap.Logger) { l.Info("ready") }),
		).Build(),
	)
	if err := app.Start(nil); err != nil {
		log.Fatal(err)
	}
}
```

## Concepts

- **Node**: a declarative unit like `Provide`, `Invoke`, `Module`, `Replace`.
- **App**: a tree of nodes. `di.App(...).Build()` returns an `fx.Option`.
- **Plan**: `di.Plan(...)` builds a tree view of what will be wired.

## Core API (high level)

- `di.App(...)` build graph into Fx options
- `di.Module(name, ...)` module scope
- `di.Provide(fn, ...)` constructor
- `di.Supply(value, ...)` value
- `di.Invoke(fn, ...)` invoke function
- `di.Decorate(fn, ...)` decorate existing values
- `di.Populate(&target, ...)` populate values (supports `di.Name`/`di.Group`)

Basic usage of the most common nodes.

```go
app := di.App(
	di.Module("core",
		di.Provide(NewService),
		di.Supply(Config{Port: 9000}),
		di.Decorate(func(s *Service) *Service { return s }),
	),
	di.Invoke(func(s *Service) {}),
	di.Populate(&someService),
).Build()
```

## Conditional wiring

Gate providers based on environment and select defaults.

```go
app := di.App(
	di.If(os.Getenv("APP_ENV") != "prod",
		di.Provide(zap.NewDevelopment),
	),
	di.Switch(
		di.WhenCase(func() bool { return os.Getenv("APP_ENV") == "prod" },
			di.Provide(zap.NewProduction),
		),
		di.DefaultCase(
			di.Provide(zap.NewExample),
		),
	),
).Build()
```

## Replace / Default

Override a provided implementation with a test/mock value.

```go
app := di.App(
	di.Provide(func() *realReader { return &realReader{} }, di.As[Reader]()),
	di.Replace(&mockReader{}, di.As[Reader]()),
).Build()
```

## Config (Viper)

### Load a TOML file

Load a config file into a typed struct.

```go
fx.New(
	di.App(
		di.Config[Config](
			"di/examples/config_toml/config.toml",
			di.ConfigType("toml"),
		),
		di.Invoke(func(cfg Config) {
			log.Println(cfg.App.Name, cfg.Db.Host)
		}),
	).Build(),
).Run()
```

### Env override

Override config values with environment variables.

```go
fx.New(
	di.App(
		di.Config[Config](
			"di/examples/config_env/config.toml",
			di.ConfigEnvOverride(),
		),
		di.Invoke(func(cfg Config, l *zap.Logger) {
			l.Info("app", zap.String("name", cfg.App.Name))
		}),
	).Build(),
).Run()
```

### Watch & restart

Watch config changes and restart the app when they change.

```go
err := di.App(
	di.ConfigFile("di/examples/config_watch/config.toml", di.ConfigType("toml")),
	di.Config[AppConfig]("app", di.ConfigWatch()),
	di.Invoke(func(cfg AppConfig) {
		log.Println("config", cfg.Name, cfg.Port)
	}),
).Run()
```

## Auto-group

Automatically collect implementations into a group.

```go
fx.New(
	di.App(
		di.AutoGroup[Handler]("handlers"),
		di.Provide(NewA),
		di.Provide(NewB),
		di.Invoke(func(handlers []Handler) {
			for _, h := range handlers {
				h.Handle()
			}
		}, di.Group("handlers")),
	).Build(),
).Run()
```

## Diagnostics

Enable source-aware error diagnostics.

```go
fx.New(
	di.App(
		di.Diagnostics(),
		di.Invoke(func(msg string) { log.Println(msg) }),
	).Build(),
).Run()
```

## Populate

Populate local variables from the container (supports name/group tags).

```go
var logger *zap.Logger
var cfg AppConfig

fx.New(
	di.App(
		di.Provide(zap.NewDevelopment, di.Name("dev")),
		di.Supply(AppConfig{Name: "demo", Port: 9000}),
		di.Populate(&logger, di.Name("dev")),
		di.Populate(&cfg),
	).Build(),
).Run()
```

## Examples

Full examples live under `di/examples`:

- `basic`
- `module`, `module2`
- `replace`, `replace2`
- `decorate`
- `if`, `switch`
- `plan`
- `populate`
- `auto_group`
- `diagnostics*`
- `config_*`

Run one:

Run an example directly.

```
go run ./di/examples/basic/main.go
```
