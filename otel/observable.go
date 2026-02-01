package otel

import (
	"log"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// ObserversGroupName is the DI group name for observable setters.
const ObserversGroupName = "observers"

// ObserverSetter is implemented by types that can receive an Observability instance.
type ObserverSetter interface {
	SetObserver(*Observer)
}

// Observable can be embedded to receive an Observability via DI auto-grouping.
type Observable struct {
	Layer string
	Obs   *Observer
}

// SetObserver stores the shared Observability instance.
func (o *Observable) SetObserver(obs *Observer) {
	if obs == nil {
		o.Obs = NewNopObserver()
		return
	}
	o.Obs = obs
}

func NewNopObserver() *Observer {
	return NewObserver(zap.NewNop(), noop.NewTracerProvider().Tracer(""))
}

// NewDefaultObserver creates an Observability using the app logger and tracer provider.
func NewDefaultObserver(logger *zap.Logger, tp *TracerProvider) *Observer {
	return NewObserver(logger, tp.Tracer(""))
}

func RegisterObservers(logger *zap.Logger, tp *TracerProvider, setters ...ObserverSetter) {
	for _, setter := range setters {
		meta, ok := di.ReflectMetadata[[]any](setter)
		if !ok || len(meta) == 0 {
			continue
		}
		layer, ok := meta[0].(LayerMetadata)
		if !ok {
			continue
		}
		log.Println("Registering observer for", layer.Name, "with setter", setter)
		setter.SetObserver(NewObserver(logger, tp.Tracer(layer.Name)))
	}
}
