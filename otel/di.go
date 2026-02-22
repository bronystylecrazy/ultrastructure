package otel

import (
	"github.com/bronystylecrazy/ultrastructure/cfg"
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func Module() di.Node {
	return di.Options(
		// auto group for otel.Telemetry
		di.AutoGroup[Observable](ObservablesGroupName),

		// config
		cfg.Config[Config]("otel", cfg.WithSourceFile("config.toml"), cfg.WithType("toml"), cfg.WithViper(applyOTELenv)),

		// observables
		di.Provide(NewLogger),
		di.Provide(NewSlog),
		di.Provide(NewResource),
		di.Provide(NewLogExporter),
		di.Provide(NewLoggerProvider),
		di.Provide(NewTraceExporter),
		di.Provide(NewTracerProvider),
		di.Provide(NewMetricExporter),
		di.Provide(NewMeterProvider),
		di.Provide(NewRuntimeMetrics, Layer("runtime")),
		di.Provide(NewAttachedObservables, di.Params(``, ``, ``, ``, di.Group(ObservablesGroupName))),

		// fx options
		fx.WithLogger(NewEventLogger),
	)
}
