package otel

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
)

func newRecordingObserver(t *testing.T) (*Observer, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return NewObserver(zap.NewNop(), tp.Tracer("test")), recorder
}

func endedSpanNames(recorder *tracetest.SpanRecorder) []string {
	var names []string
	for _, span := range recorder.Ended() {
		names = append(names, span.Name())
	}
	return names
}

func assertSpanNames(t *testing.T, recorder *tracetest.SpanRecorder, want ...string) {
	t.Helper()
	got := endedSpanNames(recorder)
	if len(got) != len(want) {
		t.Fatalf("unexpected spans: got %q want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected spans: got %q want %q", got, want)
		}
	}
}

type spanNamingService struct {
	obs *Observer
}

func (s *spanNamingService) Encode(ctx context.Context) {
	_, span := s.obs.Start(ctx)
	span.End()
}

func (s spanNamingService) Decode(ctx context.Context) {
	ctx, span := s.obs.Start(ctx)
	defer span.End()
	_, child := span.Start(ctx)
	child.End()
}

func startFromContext(ctx context.Context) {
	_, span := Start(ctx)
	span.End()
}

func TestObserverStartNamesSpanAfterPointerReceiverMethod(t *testing.T) {
	obs, recorder := newRecordingObserver(t)

	(&spanNamingService{obs: obs}).Encode(context.Background())

	assertSpanNames(t, recorder, "otel.spanNamingService.Encode")
}

func TestSpanStartNamesChildSpanAfterCaller(t *testing.T) {
	obs, recorder := newRecordingObserver(t)

	spanNamingService{obs: obs}.Decode(context.Background())

	assertSpanNames(t, recorder, "otel.spanNamingService.Decode", "otel.spanNamingService.Decode")
	parent, child := recorder.Ended()[1], recorder.Ended()[0]
	if child.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Fatalf("child span is not parented to the enclosing span")
	}
}

func TestStartNamesSpanAfterCallerUsingObserverFromContext(t *testing.T) {
	obs, recorder := newRecordingObserver(t)

	startFromContext(obs.With(context.Background()))

	assertSpanNames(t, recorder, "otel.startFromContext")
}

func TestObserverStartNamesSpanAfterClosure(t *testing.T) {
	obs, recorder := newRecordingObserver(t)

	handler := func(ctx context.Context) {
		_, span := obs.Start(ctx)
		span.End()
	}
	handler(context.Background())

	assertSpanNames(t, recorder, "otel.TestObserverStartNamesSpanAfterClosure.func1")
}

func TestObserverSpanKeepsExplicitName(t *testing.T) {
	obs, recorder := newRecordingObserver(t)

	_, span := obs.Span(context.Background(), "thumbhash.encode")
	span.End()

	assertSpanNames(t, recorder, "thumbhash.encode")
}

func TestSpanNameFromFunction(t *testing.T) {
	tests := map[string]string{
		"github.com/acme/app/imgutil.(*Service).Encode": "imgutil.Service.Encode",
		"github.com/acme/app/imgutil.Service.Decode":    "imgutil.Service.Decode",
		"github.com/acme/app/imgutil.Encode":            "imgutil.Encode",
		"main.(*apiService).Handle.func1":               "main.apiService.Handle.func1",
		"github.com/acme/app/store.(*Repo[...]).Get":    "store.Repo[...].Get",
		"gopkg.in/yaml.v3.Unmarshal":                    "yaml.v3.Unmarshal",
		"":                                              "unknown",
	}
	for function, want := range tests {
		if got := spanNameFromFunction(function); got != want {
			t.Errorf("spanNameFromFunction(%q) = %q, want %q", function, got, want)
		}
	}
}
