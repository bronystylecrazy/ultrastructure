# di

Declarative wiring for Uber Fx. `di` is built for teams that want consistent
composition patterns, easy graph introspection, and clean test overrides without
hand-editing Fx options.

== Installation ==

```
go get github.com/bronystylecrazy/ultrastructure/di
```

== How We Use It ==

* Build the graph with small, composable nodes (`Provide`, `Invoke`, `Module`).
* Keep wiring close to the boundary (modules), keep business logic in constructors.
* Use `Plan` during reviews to see what gets wired and why.
* Prefer `Replace`/`Default` for tests over ad-hoc conditional wiring.

== Core Vocabulary ==

* **Node** – a declarative unit such as `Provide`, `Invoke`, `Module`, `Replace`.
* **App** – a tree of nodes. `di.App(...).Build()` returns an `fx.Option`.
* **Plan** – a structured view of the wiring graph (`di.Plan(...)`).

== Quick Start ==

```go
package main

import (
	"fmt"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

type Store struct {
	Ready bool
}

func NewStore() *Store { return &Store{} }
func MarkStoreReady(s *Store) *Store {
	// Decorators can mutate or replace instances.
	s.Ready = true
	return s
}
func PrintStoreReady(s *Store) {
	// Invokes run after the graph is built.
	fmt.Println("store ready:", s.Ready)
}

func main() {
	app := fx.New(
		di.App(
			di.Provide(NewStore),
			di.Decorate(MarkStoreReady),
			di.Invoke(PrintStoreReady),
		).Build(),
	)
	_ = app.Start(nil)
}
```

Result of execution:

```
store ready: true
```

== Common Patterns ==

=== Param Tags ===

Use `Params` to tag positional parameters on `Provide`, `Invoke`, and `Decorate`.

```go
type Service struct {
	DB string
}

func NewPrimaryDB() string { return "primary" }
func NewTestDB() string    { return "test" }
func NewService(db string) *Service {
	// Params let you control which named input is used.
	return &Service{DB: db}
}
func PrintServiceDB(s *Service) {
	// Simple verification output.
	fmt.Println("service db:", s.DB)
}

// imports: fmt
di.App(
	di.Provide(NewPrimaryDB, di.Name("primary")),
	di.Provide(NewTestDB, di.Name("test")),
	di.Provide(NewService, di.Params(di.Name("test"))),
	di.Invoke(PrintServiceDB),
).Build()
```

Result of execution:

```
service db: test
```

=== Replace & Default ===

Swap implementations without reshaping your graph.

```go
type Reader interface {
	Read() string
}

type realReader struct{}
func (realReader) Read() string { return "real" }

type mockReader struct{}
func (mockReader) Read() string { return "mock" }

func NewReader() Reader { return realReader{} }
func PrintReader(r Reader) {
	// Replace swaps the implementation.
	fmt.Println("read:", r.Read())
}

// imports: fmt
di.App(
	di.Provide(NewReader),
	di.Replace(mockReader{}),
	di.Invoke(PrintReader),
).Build()
```

Result of execution:

```
read: mock
```

Use `ReplaceBefore` / `ReplaceAfter` when ordering matters across modules.

```go
func NewProdLabel() string { return "prod" }
func NewDevLabel() string  { return "dev" }
func PrintLabels(prod string, dev string) {
	// ReplaceBefore/After are order-sensitive.
	fmt.Println("prod:", prod, "dev:", dev)
}

// imports: fmt
di.App(
	di.Provide(NewProdLabel, di.Name("prod")),
	di.ReplaceBefore("nop", di.Name("prod")),
	di.Provide(NewDevLabel, di.Name("dev")),
	di.ReplaceAfter("json", di.Name("dev")),
	di.Invoke(PrintLabels, di.Params(di.Name("prod"), di.Name("dev"))),
).Build()
```

Result of execution:

```
prod: nop dev: json
```

=== Auto-Group ===

Collect implementations into a group without explicit `Group(...)` on each provider.

```go
type Handler interface {
	Name() string
}

type handlerA struct{}
func (handlerA) Name() string { return "A" }

type handlerB struct{}
func (handlerB) Name() string { return "B" }

func NewHandlerA() Handler { return handlerA{} }
func NewHandlerB() Handler { return handlerB{} }
func PrintHandlers(handlers []Handler) {
	// AutoGroup collects implementations into a slice.
	for _, h := range handlers {
		fmt.Println("handler:", h.Name())
	}
}

// imports: fmt
fx.New(
	di.App(
		di.AutoGroup[Handler]("handlers"),
		di.Provide(NewHandlerA),
		di.Provide(NewHandlerB),
		di.Invoke(PrintHandlers, di.Group("handlers")),
	).Build(),
).Run()
```

Result of execution:

```
handler: A
handler: B
```

=== Populate ===

Populate local variables from the container (supports name/group tags).

```go
type AppConfig struct {
	Name string
	Port int
}

var cfg AppConfig

func PrintConfig(cfg AppConfig) {
	// Populate writes into local variables after build.
	fmt.Println("config:", cfg.Name, cfg.Port)
}

// imports: context, fmt
app := fx.New(
	di.App(
		di.Supply(AppConfig{Name: "demo", Port: 9000}),
		di.Populate(&cfg),
	).Build(),
)
_ = app.Start(context.Background())
PrintConfig(cfg)
```

Result of execution:

```
config: demo 9000
```

== Config (Viper) ==

Load a config file into a typed struct.

```go
type Config struct {
	App struct {
		Name string
	}
	Db struct {
		Host string
	}
}

func PrintConfigSummary(cfg Config) {
	// Config binds file values to a typed struct.
	log.Println(cfg.App.Name, cfg.Db.Host)
}

// imports: log
fx.New(
	di.App(
		di.Config[Config](
			"di/examples/config_toml/config.toml",
			di.ConfigType("toml"),
		),
		di.Invoke(PrintConfigSummary),
	).Build(),
).Run()
```

Result of execution:

```
demo-app db.local
```

Override with environment variables.

```go
type Config struct {
	App struct {
		Name string
	}
}

func LogConfigName(cfg Config, l *zap.Logger) {
	// Env overrides land in the same struct.
	l.Info("app", zap.String("name", cfg.App.Name))
}

// imports: go.uber.org/zap
fx.New(
	di.App(
		di.Config[Config](
			"di/examples/config_env/config.toml",
			di.ConfigEnvOverride(),
		),
		di.Invoke(LogConfigName),
	).Build(),
).Run()
```

Result of execution:

```
{"level":"info","msg":"app","name":"demo-app"}
```

Watch and restart on changes.

```go
type AppConfig struct {
	Name string
	Port int
}

func PrintWatchConfig(cfg AppConfig) {
	// Watch emits on changes; this prints current values.
	log.Println("config", cfg.Name, cfg.Port)
}

// imports: log
err := di.App(
	di.ConfigFile("di/examples/config_watch/config.toml", di.ConfigType("toml")),
	di.Config[AppConfig]("app", di.ConfigWatch()),
	di.Invoke(PrintWatchConfig),
).Run()
```

Result of execution:

```
config demo-app 9000
```

== Diagnostics ==

Enable source-aware error diagnostics.

```go
func PrintDiagnostics(msg string) {
	// Diagnostics surfaces source-aware errors.
	log.Println(msg)
}

// imports: log
fx.New(
	di.App(
		di.Diagnostics(),
		di.Invoke(PrintDiagnostics),
	).Build(),
).Run()
```

Result of execution:

```
missing type: string (required by Invoke)
```

== API Reference ==

=== App and Graph ===

* `App(nodes ...any)` — build a node tree; call `.Build()` to get `fx.Option`.
* `Run(nodes ...any)` — build and run an app in one call.
* `Plan(nodes ...any)` — render a plan string for review/debugging.

=== Wiring Nodes ===

* `Provide(constructor any, opts ...any)` — register a constructor.
* `Supply(value any, opts ...any)` — register a concrete value.
* `Invoke(function any, opts ...Option)` — run a function on app start.
* `Decorate(function any, opts ...Option)` — transform an existing value.
* `Populate(args ...any)` — fill local variables from the container.
* `Module(name string, nodes ...any)` — create a named module.
* `Options(items ...any)` — group nodes without a name, or group options inside providers.
* `Replace(value any, opts ...any)` — override a value/provider.
* `ReplaceBefore(value any, opts ...any)` — override only items declared before.
* `ReplaceAfter(value any, opts ...any)` — override only items declared after.
* `Default(value any, opts ...any)` — supply a fallback when missing.

=== Conditionals ===

* `If(cond bool, nodes ...any)` — include nodes when a boolean is true.
* `When(fn any, nodes ...any)` — include nodes when `fn()` returns true.
* `Switch(items ...any)` — choose between cases and defaults.
* `Case(cond bool, nodes ...any)` — static switch case.
* `WhenCase(fn any, nodes ...any)` — dynamic switch case.
* `DefaultCase(nodes ...any)` — switch fallback.

=== Lifecycle ===

* `OnStart(fn any)` — register an Fx OnStart hook.
* `OnStop(fn any)` — register an Fx OnStop hook.

=== Auto-Group / Auto-Inject ===

* `AutoGroup[T](group ...string)` — collect implementations into a group.
* `AutoGroupFilter(fn func(reflect.Type) bool)` — filter auto-group targets.
* `AutoGroupAsSelf()` — include the concrete type when auto-grouping.
* `AutoGroupIgnoreType[T](group ...string)` — ignore auto-group for a type.
* `AutoGroupIgnore()` — ignore auto-group for a specific provider.
* `AutoInject()` — enable auto field injection for the scope.
* `AutoInjectIgnore()` — disable auto field injection for a provider.

=== Tagging and Exports ===

* `As[T](tags ...string)` — export a provider as interface/type.
* `Name(name string)` — name a value.
* `Group(name string)` — group a value.
* `ToGroup(name string)` — send the most recent `As` to a group.
* `Self()` / `AsSelf()` — export the concrete type alongside others.
* `Private()` / `Public()` — hide or expose providers across modules.
* `Params(items ...any)` — apply positional param tags.
* `InTag(tag string)` — raw param tag.
* `InGroup(name string)` — param group tag.
* `Optional()` — mark a param optional.

=== Config (Viper) ===

* `Config[T](pathOrKey string, opts ...any)` — load config into a struct.
* `ConfigFile(path string, opts ...any)` — register a config file source.
* `ConfigBind[T](key string, opts ...any)` — bind a key into a struct.
* `ConfigType(kind string)` — set file type (toml/json/yaml).
* `ConfigName(name string)` — name a config source.
* `ConfigPath(path string)` — add search paths.
* `ConfigOptional()` — do not error on missing config.
* `ConfigEnvPrefix(prefix string)` — environment prefix.
* `ConfigEnvKeyReplacer(replacer *strings.Replacer)` — env key mapping.
* `ConfigEnvOverride(prefix ...string)` — enable env overrides.
* `ConfigAutomaticEnv()` — viper automatic env.
* `ConfigNoEnv()` — disable env handling.
* `ConfigDefault(key string, value any)` — set default.
* `ConfigWithViper(fn func(*viper.Viper) error)` — customize viper instance.
* `ConfigWatch(opts ...ConfigWatchOption)` — watch config changes.
* `ConfigWatchDebounce(d time.Duration)` — debounce watch events.
* `ConfigWatchKeys(keys ...string)` — watch specific keys.
* `ConfigDisableWatch()` — disable watching for a scope.

=== Diagnostics ===

* `Diagnostics()` — enable source-aware diagnostics.

=== Utilities ===

* `ConvertAnys(nodes []Node)` — convert a list of nodes into `[]any`.

=== Exported Types ===

* `Node` — interface implemented by all nodes.
* `Option` — interface for provider/invoke options.
* `ConfigWatchOption` — interface for config watch options.

== Examples ==

Look in `di/examples` for runnable programs:

* `basic`
* `module`, `module2`
* `replace`, `replace2`
* `decorate`
* `if`, `switch`
* `plan`
* `populate`
* `auto_group`
* `diagnostics*`
* `config_*`

```
go run ./di/examples/basic/main.go
```
