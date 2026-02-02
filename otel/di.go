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
		di.Config[Config]("otel"),
		di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),

		// observables
		di.Provide(NewLogger),
		di.Provide(NewResource),
		di.Provide(NewLogExporter),
		di.Provide(NewLoggerProvider),
		di.Provide(NewTraceExporter),
		di.Provide(NewTracerProvider),
		di.Invoke(AttachTelemetryToObservables, di.Params(``, ``, di.Group(ObservablesGroupName))),

		// fx options
		fx.WithLogger(NewEventLogger),
	)
}
