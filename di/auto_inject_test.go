package di

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx"
)

type aiDep struct {
	val string
}

type aiTarget struct {
	Dep aiDep `di:"inject"`
}

type aiTargetPtr struct {
	Dep *aiDep `di:"inject"`
}

type aiNamed struct {
	val string
}

type aiGroupItem struct {
	id int
}

type aiTargetTags struct {
	Named *aiNamed     `di:"name=prod"`
	Group []aiGroupItem `di:"group=items"`
}

type aiOptional struct {
	Dep *aiDep `di:"optional"`
}

func TestAutoInjectProvideStruct(t *testing.T) {
	var got aiTarget
	app := fx.New(
		App(
			AutoInject(),
			Supply(aiDep{val: "dep"}),
			Provide(func() aiTarget { return aiTarget{} }),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got.Dep.val != "dep" {
		t.Fatalf("unexpected dep: %#v", got.Dep)
	}
}

func TestAutoInjectProvidePointer(t *testing.T) {
	var got *aiTargetPtr
	app := fx.New(
		App(
			AutoInject(),
			Supply(&aiDep{val: "dep"}),
			Provide(func() *aiTargetPtr { return &aiTargetPtr{} }),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got == nil || got.Dep == nil || got.Dep.val != "dep" {
		t.Fatalf("unexpected dep: %#v", got)
	}
}

func TestAutoInjectTagsNameAndGroup(t *testing.T) {
	var got aiTargetTags
	app := fx.New(
		App(
			AutoInject(),
			Supply(&aiNamed{val: "prod"}, Name("prod")),
			Supply(aiGroupItem{id: 1}, Group("items")),
			Supply(aiGroupItem{id: 2}, Group("items")),
			Provide(func() aiTargetTags { return aiTargetTags{} }),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got.Named == nil || got.Named.val != "prod" {
		t.Fatalf("unexpected named: %#v", got.Named)
	}
	if len(got.Group) != 2 {
		t.Fatalf("expected 2 group items, got %d", len(got.Group))
	}
}

func TestAutoInjectOptionalMissing(t *testing.T) {
	var got aiOptional
	app := fx.New(
		App(
			AutoInject(),
			Provide(func() aiOptional { return aiOptional{} }),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got.Dep != nil {
		t.Fatalf("expected nil optional dep, got %#v", got.Dep)
	}
}

func TestAutoInjectIgnoreOption(t *testing.T) {
	var got aiTarget
	app := fx.New(
		App(
			AutoInject(),
			Supply(aiDep{val: "dep"}),
			Provide(func() aiTarget { return aiTarget{} }, AutoInjectIgnore()),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got.Dep.val != "" {
		t.Fatalf("expected ignored inject, got %#v", got.Dep)
	}
}

func TestAutoInjectSupply(t *testing.T) {
	var got *aiTargetPtr
	app := fx.New(
		App(
			AutoInject(),
			Supply(&aiDep{val: "dep"}),
			Supply(&aiTargetPtr{}),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got == nil || got.Dep == nil || got.Dep.val != "dep" {
		t.Fatalf("unexpected dep: %#v", got)
	}
}

type aiHandler interface {
	Name() string
}

type aiHandlerImpl struct {
	name string
}

func (h *aiHandlerImpl) Name() string { return h.name }

type aiTargetGroup struct {
	Handlers []aiHandler `di:"group=handlers"`
}

func TestAutoInjectWithAutoGroup(t *testing.T) {
	var got aiTargetGroup
	app := fx.New(
		App(
			AutoInject(),
			AutoGroup[aiHandler]("handlers"),
			Supply(&aiHandlerImpl{name: "h1"}),
			Provide(func() aiTargetGroup { return aiTargetGroup{} }),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if len(got.Handlers) != 1 || got.Handlers[0].Name() != "h1" {
		t.Fatalf("unexpected handlers: %#v", got.Handlers)
	}
}

type aiOutResult struct {
	fx.Out
	Dep aiDep `di:"inject" name:"out"`
}

func TestAutoInjectWithFxOut(t *testing.T) {
	var got aiDep
	app := fx.New(
		App(
			AutoInject(),
			Supply(aiDep{val: "dep"}),
			Provide(func() aiOutResult {
				return aiOutResult{}
			}),
			Populate(&got, Name("out")),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got.val != "dep" {
		t.Fatalf("unexpected dep: %#v", got)
	}
}

type aiModuleTarget struct {
	Dep aiDep `di:"inject"`
}

type aiModuleTarget2 struct {
	Dep aiDep `di:"inject"`
}

func TestAutoInjectModuleInheritance(t *testing.T) {
	var got aiModuleTarget
	app := fx.New(
		App(
			AutoInject(),
			Supply(aiDep{val: "dep"}),
			Module("child",
				Provide(func() aiModuleTarget { return aiModuleTarget{} }),
			),
			Populate(&got),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if got.Dep.val != "dep" {
		t.Fatalf("unexpected dep: %#v", got.Dep)
	}
}

func TestAutoInjectModuleDoesNotLeak(t *testing.T) {
	var inModule aiModuleTarget
	var outside aiModuleTarget2
	app := fx.New(
		App(
			Supply(aiDep{val: "dep"}),
			Module("child",
				AutoInject(),
				Provide(func() aiModuleTarget { return aiModuleTarget{} }),
				Populate(&inModule),
			),
			Provide(func() aiModuleTarget2 { return aiModuleTarget2{} }),
			Populate(&outside),
		).Build(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	if inModule.Dep.val != "dep" {
		t.Fatalf("unexpected in-module dep: %#v", inModule.Dep)
	}
	if outside.Dep.val != "" {
		t.Fatalf("expected no injection outside module, got %#v", outside.Dep)
	}
}
