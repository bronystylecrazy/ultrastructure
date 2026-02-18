package us_test

import (
	"context"
	"testing"

	us "github.com/bronystylecrazy/ultrastructure"
	"github.com/bronystylecrazy/ultrastructure/cmd"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/fx"
)

type layerProbeObservable struct {
	otel.Telemetry
}

func newLayerProbeObservable() *layerProbeObservable {
	return &layerProbeObservable{Telemetry: otel.Nop()}
}

type earlierOrderedProbeObservable struct {
	otel.Telemetry
}

func newEarlierOrderedProbeObservable() *earlierOrderedProbeObservable {
	return &earlierOrderedProbeObservable{Telemetry: otel.Nop()}
}

type normalOrderedProbeObservable struct {
	otel.Telemetry
}

func newNormalOrderedProbeObservable() *normalOrderedProbeObservable {
	return &normalOrderedProbeObservable{Telemetry: otel.Nop()}
}

type laterOrderedProbeObservable struct {
	otel.Telemetry
}

func newLaterOrderedProbeObservable() *laterOrderedProbeObservable {
	return &laterOrderedProbeObservable{Telemetry: otel.Nop()}
}

func assertLayerProbeSpan(t *testing.T, expectedLayer string, opts ...any) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := &otel.TracerProvider{TracerProvider: sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)}

	app := ustest.New(t,
		di.Replace(tp),
		di.Provide(newLayerProbeObservable, opts...),
		di.Invoke(func(_ otel.Attached, probe *layerProbeObservable) {
			_, span := probe.Obs.Start(context.Background(), "layer-probe-span")
			span.End()
		}),
	)
	defer app.RequireStart().RequireStop()

	found := false
	for _, span := range recorder.Ended() {
		if span.Name() != "layer-probe-span" {
			continue
		}
		found = true
		if got, want := span.InstrumentationScope().Name, expectedLayer; got != want {
			t.Fatalf("unexpected tracer layer: got %q want %q", got, want)
		}
	}
	if !found {
		t.Fatal("expected layer-probe-span to be recorded")
	}
}

func TestNewAppliesLayerMetadataWhenPriorityComesFirst(t *testing.T) {
	assertLayerProbeSpan(t, "us.test.layer", web.Priority(web.Earlier), otel.Layer("us.test.layer"))
}

func TestNewAppliesLayerMetadataWhenLayerComesFirst(t *testing.T) {
	assertLayerProbeSpan(t, "us.test.layer", otel.Layer("us.test.layer"), web.Priority(web.Earlier))
}

func TestNewAppliesLastLayerMetadataWhenMultipleLayersProvided(t *testing.T) {
	assertLayerProbeSpan(
		t,
		"us.test.layer.v2",
		otel.Layer("us.test.layer.v1"),
		web.Priority(web.Earlier),
		otel.Layer("us.test.layer.v2"),
	)
}

func TestNewOrdersMultipleObservablesByPriority(t *testing.T) {
	var orderedIDs []string

	app := ustest.New(t,
		di.Provide(
			newLaterOrderedProbeObservable,
			otel.Layer("layer.later"),
			di.Priority(di.Later),
		),
		di.Provide(
			newEarlierOrderedProbeObservable,
			di.Priority(di.Earlier),
			otel.Layer("layer.earlier"),
		),
		di.Provide(
			newNormalOrderedProbeObservable,
			otel.Layer("layer.normal"),
		),
		di.Invoke(func(observables []otel.Observable) {
			for _, observable := range observables {
				switch observable.(type) {
				case *earlierOrderedProbeObservable:
					orderedIDs = append(orderedIDs, "earlier")
				case *normalOrderedProbeObservable:
					orderedIDs = append(orderedIDs, "normal")
				case *laterOrderedProbeObservable:
					orderedIDs = append(orderedIDs, "later")
				}
			}
		}, di.Params(di.Group(otel.ObservablesGroupName))),
	)
	defer app.RequireStart().RequireStop()

	if len(orderedIDs) != 3 {
		t.Fatalf("expected 3 ordered probe observables, got %d (%v)", len(orderedIDs), orderedIDs)
	}
	index := map[string]int{}
	for i, id := range orderedIDs {
		index[id] = i
	}
	if !(index["earlier"] < index["normal"] && index["normal"] < index["later"]) {
		t.Fatalf("unexpected priority order: %v", orderedIDs)
	}
}

func TestNewAcceptsNodeSlice(t *testing.T) {
	var got string
	nodes := []di.Node{
		di.Supply("value"),
		di.Populate(&got),
	}

	app := ustest.New(t, nodes)
	defer app.RequireStart().RequireStop()

	if got != "value" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestNewRun(t *testing.T) {
	nodes := []di.Node{
		di.Invoke(func(shutdowner fx.Shutdowner) {
			_ = shutdowner.Shutdown()
		}),
	}

	if err := us.New(nodes).Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestNewProvidesDefaultModules(t *testing.T) {
	var root *cmd.Root
	var router fiber.Router

	app := ustest.New(t,
		di.Populate(&root),
		di.Populate(&router),
	)
	defer app.RequireStart().RequireStop()

	if root == nil {
		t.Fatal("expected cmd root from default modules")
	}
	if router == nil {
		t.Fatal("expected fiber router from default modules")
	}
}
