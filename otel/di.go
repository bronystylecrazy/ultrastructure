package otel

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func Module() di.Node {
	return di.Options(
		// auto group for otel.Telemetry
		di.AutoGroup[Observable](ObservablesGroupName),

		// config
		di.Config[Config]("otel", di.ConfigWithViper(applyOTELenv)),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),

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
		di.Provide(AttachTelemetryToObservables, di.Params(``, ``, ``, ``, di.Group(ObservablesGroupName))),

		// fx options
		fx.WithLogger(NewEventLogger),
	)
}
