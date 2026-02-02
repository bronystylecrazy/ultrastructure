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

// Attached is a marker indicating telemetry has been wired.
type Attached struct{}

// apply stores the shared Observability instance.
func (o *Telemetry) apply(obs *Observer) {
	if obs == nil {
		o.Obs = NopObserver()
		return
	}
	o.Obs = obs
}

func Nop() Telemetry {
	return Telemetry{Obs: NopObserver()}
}

func NopObserver() *Observer {
	return NewObserver(zap.NewNop(), noop.NewTracerProvider().Tracer(""))
}

func AttachTelemetryToObservables(logger *zap.Logger, tp *TracerProvider, observables ...Observable) Attached {
	for _, observable := range observables {
		meta, ok := di.ReflectMetadata[[]any](observable)
		if !ok || len(meta) == 0 {
			continue
		}
		layer, ok := meta[0].(LayerMetadata)
		if !ok {
			continue
		}

		obs := NewObserver(logger.With(zap.String("app.layer", layer.Name)), tp.Tracer(layer.Name))
		obs.layerName = layer.Name
		observable.apply(obs)
	}
	return Attached{}
}
