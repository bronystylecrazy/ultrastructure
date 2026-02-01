package otel

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// ObservablesGroupName is the DI group name for observable setters.
const ObservablesGroupName = "us.observables"

// Observable is implemented by types that can receive an Observability instance.
type Observable interface {
	apply(*Observer)
}

// Observable can be embedded to receive an Observability via DI auto-grouping.
type Telemetry struct {
	Obs *Observer
}

// apply stores the shared Observability instance.
func (o *Telemetry) apply(obs *Observer) {
	if obs == nil {
		o.Obs = NewNopObserver()
		return
	}
	o.Obs = obs
}

func Nop() Telemetry {
	return Telemetry{Obs: NewNopObserver()}
}

func NewNopObserver() *Observer {
	return NewObserver(zap.NewNop(), noop.NewTracerProvider().Tracer(""))
}

// NewDefaultObserver creates an Observability using the app logger and tracer provider.
func NewDefaultObserver(logger *zap.Logger, tp *TracerProvider) *Observer {
	return NewObserver(logger, tp.Tracer(""))
}

func AttachTelemetryToObservables(logger *zap.Logger, tp *TracerProvider, observables ...Observable) {
	for _, observable := range observables {
		meta, ok := di.ReflectMetadata[[]any](observable)
		if !ok || len(meta) == 0 {
			continue
		}
		layer, ok := meta[0].(LayerMetadata)
		if !ok {
			continue
		}
		observable.apply(NewObserver(logger.With(zap.String("app.layer", layer.Name)), tp.Tracer(layer.Name)))
	}
}
