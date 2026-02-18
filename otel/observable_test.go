package otel

import (
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

type testObservable struct {
	obs *Observer
}

func (t *testObservable) apply(obs *Observer) {
	t.obs = obs
}

func TestNewAttachedObservablesUsesLayerMetadataRegardlessOfOrder(t *testing.T) {
	logger := zap.NewNop()
	tp := &TracerProvider{TracerProvider: sdktrace.NewTracerProvider()}
	observable := &testObservable{}
	di.RegisterMetadata(observable, 42, LayerMetadata{Kind: "layer", Name: "web.http"})

	NewAttachedObservables(logger, tp, nil, Config{}, observable)

	if observable.obs == nil {
		t.Fatalf("expected observable to receive observer")
	}
	if observable.obs.layerName != "web.http" {
		t.Fatalf("unexpected layer name: got %q want %q", observable.obs.layerName, "web.http")
	}
}

func TestNewAttachedObservablesFallsBackToServiceWithoutLayerMetadata(t *testing.T) {
	logger := zap.NewNop()
	tp := &TracerProvider{TracerProvider: sdktrace.NewTracerProvider()}
	observable := &testObservable{}
	di.RegisterMetadata(observable, 42)

	NewAttachedObservables(logger, tp, nil, Config{}, observable)

	if observable.obs == nil {
		t.Fatalf("expected observable to receive observer")
	}
	if observable.obs.layerName != "service" {
		t.Fatalf("unexpected layer name: got %q want %q", observable.obs.layerName, "service")
	}
}

func TestNewAttachedObservablesUsesLastLayerMetadata(t *testing.T) {
	logger := zap.NewNop()
	tp := &TracerProvider{TracerProvider: sdktrace.NewTracerProvider()}
	observable := &testObservable{}
	di.RegisterMetadata(
		observable,
		LayerMetadata{Kind: "layer", Name: "web.http"},
		LayerMetadata{Kind: "layer", Name: "web.http.v2"},
	)

	NewAttachedObservables(logger, tp, nil, Config{}, observable)

	if observable.obs == nil {
		t.Fatalf("expected observable to receive observer")
	}
	if observable.obs.layerName != "web.http.v2" {
		t.Fatalf("unexpected layer name: got %q want %q", observable.obs.layerName, "web.http.v2")
	}
}
