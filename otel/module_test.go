package otel

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/ditest"
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
		Module(),
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
