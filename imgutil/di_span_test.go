package imgutil

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
)

func TestServiceSpansAreNamedAfterTheCallingMethod(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	var hasher ThumbHasher = &Service{
		Telemetry: otel.Telemetry{Obs: otel.NewObserver(zap.NewNop(), tp.Tracer("thumbhash"))},
	}

	ctx := context.Background()
	img := testFixtureImage()
	pngData := encodePNG(t, img)
	hashData := EncodeThumbHash(ctx, img)
	hashBase64 := EncodeThumbHashBase64(ctx, img)
	hash, err := DecodeHash(ctx, hashData, nil)
	if err != nil {
		t.Fatalf("decode hash fixture: %v", err)
	}
	cfg := DecodingCfg{BaseSize: 24, SaturationBoost: 1.5}
	fileHeader := newMultipartFileHeader(t, "file", "fixture.png", pngData)

	calls := map[string]func(){
		"EncodeThumbHash":                 func() { hasher.EncodeThumbHash(ctx, img) },
		"EncodeThumbHashBase64":           func() { hasher.EncodeThumbHashBase64(ctx, img) },
		"EncodeThumbHashResult":           func() { _, _ = hasher.EncodeThumbHashResult(ctx, img) },
		"EncodeThumbHashFromBytes":        func() { _, _ = hasher.EncodeThumbHashFromBytes(ctx, pngData) },
		"EncodeThumbHashBase64FromBytes":  func() { _, _ = hasher.EncodeThumbHashBase64FromBytes(ctx, pngData) },
		"EncodeThumbHashResultFromBytes":  func() { _, _ = hasher.EncodeThumbHashResultFromBytes(ctx, pngData) },
		"EncodeThumbHashFromReader":       func() { _, _ = hasher.EncodeThumbHashFromReader(ctx, bytes.NewReader(pngData)) },
		"EncodeThumbHashBase64FromReader": func() { _, _ = hasher.EncodeThumbHashBase64FromReader(ctx, bytes.NewReader(pngData)) },
		"EncodeThumbHashResultFromReader": func() { _, _ = hasher.EncodeThumbHashResultFromReader(ctx, bytes.NewReader(pngData)) },
		"DecodeThumbHash":                 func() { _, _ = hasher.DecodeThumbHash(ctx, hashData) },
		"DecodeThumbHashFromBytes":        func() { _, _ = hasher.DecodeThumbHashFromBytes(ctx, hashData) },
		"DecodeThumbHashFromBytesWithCfg": func() { _, _ = hasher.DecodeThumbHashFromBytesWithCfg(ctx, hashData, cfg) },
		"DecodeThumbHashFromReader":       func() { _, _ = hasher.DecodeThumbHashFromReader(ctx, bytes.NewReader(hashData)) },
		"DecodeThumbHashFromReaderWithCfg": func() {
			_, _ = hasher.DecodeThumbHashFromReaderWithCfg(ctx, bytes.NewReader(hashData), cfg)
		},
		"DecodeThumbHashWithCfg":       func() { _, _ = hasher.DecodeThumbHashWithCfg(ctx, hashData, cfg) },
		"DecodeThumbHashBase64":        func() { _, _ = hasher.DecodeThumbHashBase64(ctx, hashBase64) },
		"DecodeThumbHashBase64WithCfg": func() { _, _ = hasher.DecodeThumbHashBase64WithCfg(ctx, hashBase64, cfg) },
		"DecodeThumbHashResult":        func() { _, _ = hasher.DecodeThumbHashResult(ctx, hashData) },
		"DecodeThumbHashResultWithCfg": func() { _, _ = hasher.DecodeThumbHashResultWithCfg(ctx, hashData, cfg) },
		"DecodeHash":                   func() { _, _ = hasher.DecodeHash(ctx, hashData, &cfg) },
		"DecodeHashBase64":             func() { _, _ = hasher.DecodeHashBase64(ctx, hashBase64, &cfg) },
		"EncodeHash":                   func() { _, _ = hasher.EncodeHash(ctx, hash) },
		"EncodeHashBase64":             func() { _, _ = hasher.EncodeHashBase64(ctx, hash) },
		"HashSize":                     func() { _, _, _ = hasher.HashSize(ctx, hash, 32) },
		"GenerateThumbHash":            func() { _, _ = hasher.GenerateThumbHash(ctx, fileHeader) },
	}

	methods := reflect.TypeOf((*ThumbHasher)(nil)).Elem()
	if methods.NumMethod() != len(calls) {
		t.Fatalf("ThumbHasher has %d methods but %d are exercised", methods.NumMethod(), len(calls))
	}
	for i := range methods.NumMethod() {
		method := methods.Method(i).Name
		call, ok := calls[method]
		if !ok {
			t.Errorf("%s is not exercised", method)
			continue
		}
		before := len(recorder.Ended())
		call()
		ended := recorder.Ended()
		if len(ended) != before+1 {
			t.Errorf("%s ended %d spans, want 1", method, len(ended)-before)
			continue
		}
		if got, want := ended[before].Name(), "imgutil.Service."+method; got != want {
			t.Errorf("%s span name = %q, want %q", method, got, want)
		}
	}
}
