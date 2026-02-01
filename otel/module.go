package otel

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.uber.org/fx"
)

func Module() di.Node {
	return di.Options(
		di.AutoGroup[ObserverSetter](ObserversGroupName),
		di.Module(
			"us/opentelemetry",
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride()),
			di.Config[Config]("otel"),
			di.Provide(NewResource),
			di.Provide(NewLogExporter),
			di.Provide(NewLoggerProvider),
			di.Provide(NewTraceExporter),
			di.Provide(NewTracerProvider),
			di.Provide(NewDefaultObserver),
		),
		fx.Decorate(AttachLoggerToOtel),
		di.Invoke(RegisterObservers, di.Params(``, ``, di.Group(ObserversGroupName))),
	)
}
