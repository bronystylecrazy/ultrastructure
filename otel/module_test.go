package otel

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ditest"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type moduleProbeObservable struct {
	Telemetry
}

func newModuleProbeObservable() *moduleProbeObservable {
	return &moduleProbeObservable{Telemetry: Nop()}
}

func TestModuleAutoGroupsObservableAndAppliesLayer(t *testing.T) {
	var layer string
	app := ditest.New(t,
		Providers(),
		di.Supply(context.Background(), di.As[context.Context]()),
		di.Provide(newModuleProbeObservable, Layer("otel.module.test")),
		di.Invoke(func(_ Attached, probe *moduleProbeObservable) {
			if probe.Obs == nil {
				return
			}
			layer = probe.Obs.layerName
		}),
	)
	defer app.RequireStart().RequireStop()

	if layer != "otel.module.test" {
		t.Fatalf("unexpected layer: got %q want %q", layer, "otel.module.test")
	}
}

// The exporters must resolve with NOTHING beyond the module's own providers:
// registering the variadic NewLogExporter/NewTraceExporter directly made
// fx.Annotate (via the Observable auto-group) demand a required
// []otlploggrpc.Option / []otlptracegrpc.Option that no app provides, and
// because the exporters are lc.stoppers (built at start), every consumer app
// failed at startup.
func TestModuleExportersResolveWithoutOptionSlices(t *testing.T) {
	var resolved bool
	app := ditest.New(t,
		Providers(),
		di.Supply(context.Background(), di.As[context.Context]()),
		di.Invoke(func(_ sdklog.Exporter, _ sdktrace.SpanExporter) {
			resolved = true
		}),
	)
	defer app.RequireStart().RequireStop()

	if !resolved {
		t.Fatal("exporters did not resolve from the module's own providers")
	}
}
