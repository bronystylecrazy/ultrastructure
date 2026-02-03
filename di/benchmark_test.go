package di

import (
	"context"
	"testing"

	"go.uber.org/fx"
)

type benchA struct{}
type benchB struct{}
type benchC struct{}
type benchD struct{}
type benchE struct{}
type benchF struct{}
type benchG struct{}
type benchH struct{}
type benchI struct{}
type benchJ struct{}
type benchK struct{}
type benchL struct{}
type benchHugeItem struct{ ID int }
type benchDecorate struct{ Value int }
type benchReplace struct{ Value int }
type benchAutoInjectDep struct{}
type benchAutoInjectTarget struct {
	Dep *benchAutoInjectDep `di:"inject"`
}
type benchGroupIface interface {
	ID() string
}
type benchGroupImpl struct{ id string }
type benchStressItem struct{ ID int }
type benchStressHandler interface {
	ID() int
}
type benchStressHandlerImpl struct{ id int }

func newBenchA() *benchA { return &benchA{} }
func newBenchB(a *benchA) *benchB { return &benchB{} }
func newBenchC(a *benchA, b *benchB) *benchC { return &benchC{} }
func newBenchD(a *benchA, c *benchC) *benchD { return &benchD{} }
func newBenchE(b *benchB, d *benchD) *benchE { return &benchE{} }
func newBenchF(c *benchC, e *benchE) *benchF { return &benchF{} }
func newBenchG(d *benchD, f *benchF) *benchG { return &benchG{} }
func newBenchH(e *benchE, g *benchG) *benchH { return &benchH{} }
func newBenchI(f *benchF, h *benchH) *benchI { return &benchI{} }
func newBenchJ(g *benchG, i *benchI) *benchJ { return &benchJ{} }
func newBenchK(h *benchH, j *benchJ) *benchK { return &benchK{} }
func newBenchL(i *benchI, k *benchK) *benchL { return &benchL{} }
func newBenchDecorate() *benchDecorate { return &benchDecorate{Value: 1} }
func decorateBench(d *benchDecorate) *benchDecorate {
	d.Value++
	return d
}
func newBenchReplace() *benchReplace { return &benchReplace{Value: 1} }
func newBenchReplaceAlt() *benchReplace { return &benchReplace{Value: 2} }
func newBenchAutoInjectDep() *benchAutoInjectDep { return &benchAutoInjectDep{} }
func newBenchAutoInjectTarget() *benchAutoInjectTarget { return &benchAutoInjectTarget{} }
func newBenchGroupImpl() *benchGroupImpl { return &benchGroupImpl{id: "g"} }
func (b *benchGroupImpl) ID() string { return b.id }
func invokeStress(_ []benchStressItem) {}
func (b *benchStressHandlerImpl) ID() int { return b.id }
func invokeStressHandlers(_ []benchStressHandler) {}

func invokeBench(_ *benchC) {}
func invokeBenchLarge(_ *benchL) {}

func BenchmarkStartupFx(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			fx.Provide(newBenchA, newBenchB, newBenchC),
			fx.Invoke(invokeBench),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupDi(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(
				Provide(newBenchA),
				Provide(newBenchB),
				Provide(newBenchC),
				Invoke(invokeBench),
			).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupFxLarge(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			fx.Provide(
				newBenchA,
				newBenchB,
				newBenchC,
				newBenchD,
				newBenchE,
				newBenchF,
				newBenchG,
				newBenchH,
				newBenchI,
				newBenchJ,
				newBenchK,
				newBenchL,
			),
			fx.Invoke(invokeBenchLarge),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupDiLarge(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(
				Provide(newBenchA),
				Provide(newBenchB),
				Provide(newBenchC),
				Provide(newBenchD),
				Provide(newBenchE),
				Provide(newBenchF),
				Provide(newBenchG),
				Provide(newBenchH),
				Provide(newBenchI),
				Provide(newBenchJ),
				Provide(newBenchK),
				Provide(newBenchL),
				Invoke(invokeBenchLarge),
			).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkBuildFx(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fx.Options(
			fx.NopLogger,
			fx.Provide(newBenchA, newBenchB, newBenchC),
			fx.Invoke(invokeBench),
		)
	}
}

func BenchmarkBuildDi(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = App(
			Provide(newBenchA),
			Provide(newBenchB),
			Provide(newBenchC),
			Invoke(invokeBench),
		).Build()
	}
}

func BenchmarkBuildFxLarge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fx.Options(
			fx.NopLogger,
			fx.Provide(
				newBenchA,
				newBenchB,
				newBenchC,
				newBenchD,
				newBenchE,
				newBenchF,
				newBenchG,
				newBenchH,
				newBenchI,
				newBenchJ,
				newBenchK,
				newBenchL,
			),
			fx.Invoke(invokeBenchLarge),
		)
	}
}

func BenchmarkBuildDiLarge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = App(
			Provide(newBenchA),
			Provide(newBenchB),
			Provide(newBenchC),
			Provide(newBenchD),
			Provide(newBenchE),
			Provide(newBenchF),
			Provide(newBenchG),
			Provide(newBenchH),
			Provide(newBenchI),
			Provide(newBenchJ),
			Provide(newBenchK),
			Provide(newBenchL),
			Invoke(invokeBenchLarge),
		).Build()
	}
}

func buildDiLargeOption() fx.Option {
	return App(
		Provide(newBenchA),
		Provide(newBenchB),
		Provide(newBenchC),
		Provide(newBenchD),
		Provide(newBenchE),
		Provide(newBenchF),
		Provide(newBenchG),
		Provide(newBenchH),
		Provide(newBenchI),
		Provide(newBenchJ),
		Provide(newBenchK),
		Provide(newBenchL),
		Invoke(invokeBenchLarge),
	).Build()
}

func BenchmarkBuildDiLargeProfile(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = buildDiLargeOption()
	}
}

func BenchmarkBuildFxHuge(b *testing.B) {
	b.ReportAllocs()
	providers := make([]any, 0, 1000)
	for i := 0; i < 1000; i++ {
		id := i
		provider := func() benchHugeItem { return benchHugeItem{ID: id} }
		providers = append(providers, fx.Annotate(provider, fx.ResultTags(`group:"huge"`)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fx.Options(
			fx.NopLogger,
			fx.Provide(providers...),
		)
	}
}

func BenchmarkBuildDiHuge(b *testing.B) {
	b.ReportAllocs()
	nodes := make([]any, 0, 1000)
	for i := 0; i < 1000; i++ {
		id := i
		nodes = append(nodes, Provide(func() benchHugeItem { return benchHugeItem{ID: id} }, Group("huge")))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = App(nodes...).Build()
	}
}

func BenchmarkStartupFxHuge(b *testing.B) {
	b.ReportAllocs()
	providers := make([]any, 0, 1000)
	for i := 0; i < 1000; i++ {
		id := i
		provider := func() benchHugeItem { return benchHugeItem{ID: id} }
		providers = append(providers, fx.Annotate(provider, fx.ResultTags(`group:"huge"`)))
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			fx.Provide(providers...),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupDiHuge(b *testing.B) {
	b.ReportAllocs()
	nodes := make([]any, 0, 1000)
	for i := 0; i < 1000; i++ {
		id := i
		nodes = append(nodes, Provide(func() benchHugeItem { return benchHugeItem{ID: id} }, Group("huge")))
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(nodes...).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkBuildDecorate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = App(
			Provide(newBenchDecorate),
			Decorate(decorateBench),
		).Build()
	}
}

func BenchmarkBuildAutoGroup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = App(
			AutoGroup[benchGroupIface]("bench"),
			Provide(newBenchGroupImpl),
		).Build()
	}
}

func BenchmarkBuildAutoInject(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = App(
			AutoInject(),
			Provide(newBenchAutoInjectDep),
			Provide(newBenchAutoInjectTarget),
		).Build()
	}
}

func BenchmarkBuildReplace(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = App(
			Provide(newBenchReplace),
			Replace(newBenchReplaceAlt()),
		).Build()
	}
}

func BenchmarkBuildPopulate(b *testing.B) {
	b.ReportAllocs()
	var target *benchDecorate
	for i := 0; i < b.N; i++ {
		_ = App(
			Provide(newBenchDecorate),
			Populate(&target),
		).Build()
	}
}

func BenchmarkStartupDecorate(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(
				Provide(newBenchDecorate),
				Decorate(decorateBench),
			).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupAutoGroup(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(
				AutoGroup[benchGroupIface]("bench"),
				Provide(newBenchGroupImpl),
			).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupAutoInject(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(
				AutoInject(),
				Provide(newBenchAutoInjectDep),
				Provide(newBenchAutoInjectTarget),
			).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupReplace(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(
				Provide(newBenchReplace),
				Replace(newBenchReplaceAlt()),
			).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupPopulate(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		var target *benchDecorate
		app := fx.New(
			fx.NopLogger,
			App(
				Provide(newBenchDecorate),
				Populate(&target),
			).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupStressFx(b *testing.B) {
	const stressCount = 5000
	b.ReportAllocs()
	providers := make([]any, 0, stressCount)
	for i := 0; i < stressCount; i++ {
		id := i
		provider := func() benchStressItem { return benchStressItem{ID: id} }
		providers = append(providers, fx.Annotate(provider, fx.ResultTags(`group:"stress"`)))
	}
	invoke := fx.Annotate(invokeStress, fx.ParamTags(`group:"stress"`))
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			fx.Provide(providers...),
			fx.Invoke(invoke),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupStressDi(b *testing.B) {
	const stressCount = 5000
	b.ReportAllocs()
	nodes := make([]any, 0, stressCount+1)
	for i := 0; i < stressCount; i++ {
		id := i
		nodes = append(nodes, Provide(func() benchStressItem { return benchStressItem{ID: id} }, Group("stress")))
	}
	nodes = append(nodes, Invoke(invokeStress, Group("stress")))
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(nodes...).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupStressFxGroup(b *testing.B) {
	const stressCount = 5000
	b.ReportAllocs()
	providers := make([]any, 0, stressCount)
	for i := 0; i < stressCount; i++ {
		id := i
		provider := func() benchStressHandler { return &benchStressHandlerImpl{id: id} }
		providers = append(providers, fx.Annotate(provider, fx.ResultTags(`group:"stress"`)))
	}
	invoke := fx.Annotate(invokeStressHandlers, fx.ParamTags(`group:"stress"`))
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			fx.Provide(providers...),
			fx.Invoke(invoke),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}

func BenchmarkStartupStressDiAutoGroup(b *testing.B) {
	const stressCount = 5000
	b.ReportAllocs()
	nodes := make([]any, 0, stressCount+2)
	nodes = append(nodes, AutoGroup[benchStressHandler]("stress"))
	for i := 0; i < stressCount; i++ {
		id := i
		nodes = append(nodes, Provide(func() *benchStressHandlerImpl { return &benchStressHandlerImpl{id: id} }, Group("stress")))
	}
	nodes = append(nodes, Invoke(invokeStressHandlers, Group("stress")))
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := fx.New(
			fx.NopLogger,
			App(nodes...).Build(),
		)
		if err := app.Start(ctx); err != nil {
			b.Fatalf("start: %v", err)
		}
		_ = app.Stop(ctx)
	}
}
