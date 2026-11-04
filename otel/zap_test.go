package otel

import (
	"context"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
)

// memoryLogExporter keeps a copy of every exported record's body and
// attributes, rendered as strings.
type memoryLogExporter struct {
	mu      sync.Mutex
	records []map[string]string
}

func (e *memoryLogExporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range records {
		got := map[string]string{"body": renderLogValue(r.Body())}
		r.WalkAttributes(func(kv log.KeyValue) bool {
			got[strings.Clone(kv.Key)] = renderLogValue(kv.Value)
			return true
		})
		e.records = append(e.records, got)
	}
	return nil
}

func (*memoryLogExporter) Shutdown(context.Context) error   { return nil }
func (*memoryLogExporter) ForceFlush(context.Context) error { return nil }

func renderLogValue(v log.Value) string {
	switch v.Kind() {
	case log.KindString:
		return strings.Clone(v.AsString())
	case log.KindBytes:
		return string(v.AsBytes())
	case log.KindSlice:
		parts := make([]string, 0, len(v.AsSlice()))
		for _, item := range v.AsSlice() {
			parts = append(parts, renderLogValue(item))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case log.KindMap:
		parts := make([]string, 0, len(v.AsMap()))
		for _, kv := range v.AsMap() {
			parts = append(parts, kv.Key+"="+renderLogValue(kv.Value))
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		return v.String()
	}
}

type textStringer struct{ s string }

func (t textStringer) String() string { return t.s }

type nilStringer struct{}

func (*nilStringer) String() string { panic("nil receiver") }

// Fiber and fasthttp hand out strings that alias request buffers reused by the
// next request. Logs reach OpenTelemetry through a batch exported later, so
// every string the bridge keeps has to be a copy made while logging.
func TestNewLoggerExportsLoggedStringsAfterTheirBufferIsReused(t *testing.T) {
	exporter := &memoryLogExporter{}
	lp := &LoggerProvider{sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)}
	t.Cleanup(func() { _ = lp.Shutdown(context.Background()) })

	logger, err := NewLogger(Config{Enabled: true, ServiceName: "test", LogLevel: "info"}, lp)
	if err != nil {
		t.Fatal(err)
	}

	const original = "/api/v1/catalog/artifacts"
	buf := []byte(original)
	aliased := unsafe.String(&buf[0], len(buf))

	logger.With(zap.String("with", aliased)).Info(aliased,
		zap.String("string", aliased),
		zap.String(aliased, "key"),
		zap.ByteString("bytestring", buf),
		zap.Binary("binary", buf),
		zap.Stringer("stringer", textStringer{aliased}),
		zap.Reflect("reflect", aliased),
		zap.Strings("strings", []string{aliased}),
		zap.Dict("dict", zap.String("inner", aliased)),
	)

	// The next request reuses the buffer.
	copy(buf, "/api/v1/manifests/v10/xyz")

	if err := lp.ForceFlush(context.Background()); err != nil {
		t.Fatal(err)
	}

	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	if len(exporter.records) != 1 {
		t.Fatalf("exported %d records, want 1", len(exporter.records))
	}
	got := exporter.records[0]
	want := map[string]string{
		"body":       original,
		"with":       original,
		"string":     original,
		original:     "key",
		"bytestring": original,
		"binary":     original,
		"stringer":   original,
		"reflect":    original,
		"strings":    "[" + original + "]",
		"dict":       "{inner=" + original + "}",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q", key, got[key], value)
		}
	}
}

// Copying must not change how zap renders a Stringer with a nil receiver.
func TestCopyFieldLeavesNilStringerToZap(t *testing.T) {
	var s *nilStringer
	field := copyField(zap.Stringer("stringer", s))
	if field.Interface != any(s) {
		t.Fatalf("nil stringer was replaced: %#v", field)
	}
}
