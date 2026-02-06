package otel

import (
	"github.com/bronystylecrazy/ultrastructure/di"
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
type Attached struct {
	Logger         *zap.Logger
	TracerProvider *TracerProvider
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

func AttachTelemetryToObservables(logger *zap.Logger, tp *TracerProvider, observables ...Observable) Attached {
	for _, observable := range observables {
		layerName := "service"
		if meta, ok := di.ReflectMetadata[[]any](observable); ok && len(meta) > 0 {
			if layer, ok := meta[0].(LayerMetadata); ok && layer.Name != "" {
				layerName = layer.Name
			}
		}
		obs := NewObserver(logger.With(zap.String("app.layer", layerName)), tp.Tracer(layerName))
		obs.layerName = layerName
		observable.apply(obs)
	}

	logger.Debug("auto injected telemetry to observables", zap.Int("count", len(observables)))

	return Attached{
		Logger:         logger,
		TracerProvider: tp,
	}
}
