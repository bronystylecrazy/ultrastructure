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
