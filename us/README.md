# ultrastructure/us

Small helper layer on top of Uber Fx to keep DI wiring consistent across projects.
This package focuses on:
- concise `Provide`/`Invoke`/`Decorate` helpers
- grouped readers/handlers
- typed group helpers for your own interfaces
- a few quality‑of‑life wrappers (`Module`, `Options`, `Supply`, `Replace`, `Private`)

Below is a quick manual with the patterns used in this repo.

## Install

```bash
go get github.com/bronystylecrazy/ultrastructure/us
```

## Provide

`us.Provide` wraps `fx.Provide` and adds a small option system so you can expose
one constructor as multiple interfaces/groups.

```go
us.Provide(
	NewApp,
	us.AsReaderGroup(),
	us.AsHandlerGroup(),
)
```

## Bind (declarative)

The declarative API lives in `us/di` and compiles down to native Fx options.

```go
app := di.App(
	di.Provide(
		NewHandler,
		di.As[Handler](),
		di.Name("primary"),
	),
).Build()

fx.New(app).Run()
```

Grouped version:

```go
di.App(
	di.Provide(
		NewReader,
		di.As[Reader](),
		di.Group("readers"),
	),
).Build()
```

Decorate inside Provide inherits the Name/Group if you don't pass tags:

```go
di.App(
	di.Provide(
		zap.NewProduction,
		di.Name("prod"),
		di.Group("loggers"),
		di.Decorate(func(l *zap.Logger) *zap.Logger {
			return l.With(zap.String("module", "core"))
		}),
	),
).Build()
```

`di.Name` is also used for Invoke/Decorate input tags:

```go
di.App(
	di.Invoke(func(h Handler) {}, di.Name("primary")),
).Build()
```

`di.Group` also works for Invoke/Decorate:

```go
di.App(
	di.Invoke(func(rs []Reader) {}, di.Group("readers")),
).Build()
```

### Default Reader / Handler groups

```go
us.Provide(NewReader, us.AsReaderGroup())
us.Provide(NewHandler, us.AsHandlerGroup())
```

Custom group names:

```go
us.Provide(NewReader, us.AsReaderGroup("custom-readers"))
us.Provide(NewHandler, us.AsHandlerGroup("custom-handlers"))
```

### Your own groups

Two styles are supported.

Option A (group object):

```go
g := us.Group[Notifier]("notifiers")

us.Provide(NewEmailNotifier, g.As())
us.Invoke(func(xs []Notifier) {}, g.In())
```

Option B (direct helpers):

```go
us.Provide(NewEmailNotifier, us.AsGroup[Notifier]("notifiers"))
us.Invoke(func(xs []Notifier) {}, us.InGroup("notifiers"))
```

### Provide as a plain interface (no group)

Use `AsType` when you want interface typing without grouping.

```go
us.Provide(NewRepo, us.AsType[Repo]())
```

Named singletons:

```go
us.Provide(NewRepo, us.AsType[Repo]("primary"))
us.Provide(NewReader, us.AsReader("ro"))
```

### Keep the concrete type too

```go
us.Provide(NewRepo, us.AsType[Repo](), us.AsSelf())
```

### Private providers

```go
us.Module("internal",
	us.Provide(NewSecret, us.Private()),
)
```

`us.Public()` clears a previous `us.Private()` in the same call (last one wins).

## Supply

`us.Supply` mirrors `fx.Supply` and can also accept `ProvideOption`s.

```go
us.Supply(
	&MyReader{},
	us.AsReaderGroup(),
)
```

`Supply` supports `AsSelf()` as well.

## Replace

`us.Replace` mirrors `fx.Replace` and accepts **interface typing** options.
Groups are not supported by Replace (groups are append‑only).

```go
us.Replace(
	&MockReader{},
	us.AsType[Reader](),
)
```

## Invoke

`us.Invoke` wraps `fx.Invoke` and supports param tag helpers.

```go
us.Invoke(
	func(readers []us.Reader, handlers []us.Handler) {},
	us.InReaders(),
	us.InHandlers(),
)
```

Named/optional params:

```go
us.Invoke(func(db *sql.DB) {}, us.InName("ro"))
us.Invoke(func(cache *redis.Client) {}, us.InOptional())
```

### Variadic

Fx supports variadic params; use group tags the same way:

```go
us.Invoke(
	func(handlers ...us.Handler) {},
	us.InHandlers("my-handlers"),
)
```

## Decorate

`us.Decorate` wraps `fx.Decorate` and uses the same param tag helpers as `Invoke`.

```go
us.Decorate(
	func(readers []us.Reader) []us.Reader { return readers },
	us.InReaders(),
)
```

## Module / Options

```go
us.Module("core",
	us.Provide(NewLogger),
)

us.Options(
	us.Provide(NewCache),
	us.Invoke(Start),
)
```

## Plan (di)

`di.Plan` builds the declarative graph and returns a readable plan string.

```go
plan, err := di.Plan(
	di.Module("core",
		di.Provide(NewLogger, di.Name("prod")),
		di.Invoke(func(l *zap.Logger) {}),
	),
	di.Replace(zap.NewNop),
)
if err != nil {
	log.Fatal(err)
}
fmt.Println(plan)
```

## Raw Fx annotate helpers

If you prefer `fx.Annotate`, there are small helpers for tags/annotations:

```go
fx.Provide(
	fx.Annotate(NewApp, us.AsReaderAnn()...),
)

fx.Invoke(
	fx.Annotate(
		func(rs []us.Reader) {},
		fx.ParamTags(us.InReadersTag()),
	),
)
```

Available helpers:
- `AsReaderAnn`, `AsHandlerAnn`, `AsGroupAnn[T]`
- `InReadersTag`, `InHandlersTag`, `InGroupTag`
- `NameTag`, `OptionalTag`

## Examples

Examples are under `examples/`:

- `examples/basic` — mixed Provide/Invoke/Supply
- `examples/decorate` — decorate a group
- `examples/private` — private providers with modules
- `examples/replace` — replace with interface typing
- `examples/zap-decorate` — decorate `zap.Logger` across modules

Run:

```bash
go run ./examples/basic
```
