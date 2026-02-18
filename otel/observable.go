package otel

import (
	"github.com/bronystylecrazy/ultrastructure/di"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
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
	return Telemetry{
		Obs: NewNopObserver(),
	}
}

func NewAttachedObservables(logger *zap.Logger, tp *TracerProvider, mp *MeterProvider, config Config, observables ...Observable) Attached {
	defaultMetricAttrs := DefaultMetricAttributes(config)
	for _, observable := range observables {
		layerName := "service"
		if meta, ok := di.ReflectMetadata[[]any](observable); ok && len(meta) > 0 {
			for _, item := range meta {
				layer, ok := item.(LayerMetadata)
				if !ok || layer.Name == "" {
					continue
				}
				layerName = layer.Name
			}
		}
		meter := metricnoop.NewMeterProvider().Meter(layerName)
		if mp != nil && mp.MeterProvider != nil {
			meter = mp.Meter(layerName)
		}
		obs := NewObserver(logger.With(zap.String("app.layer", layerName)), tp.Tracer(layerName), meter)
		obs.layerName = layerName
		obs.defaultMetricCtx = defaultMetricAttrs
		obs.initDefaultMetricOptions()
		observable.apply(obs)
	}

	return Attached{
		Logger:         logger,
		TracerProvider: tp,
	}
}
