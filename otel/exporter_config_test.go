package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestDefaultOTLPTargetsLocalGreptime(t *testing.T) {
	config := Config{Enabled: true}

	for _, tc := range []struct {
		signal   string
		resolved OTLPConfig
		want     string
	}{
		{"traces", config.OTLPForTraces(), "http://127.0.0.1:4000/v1/otlp/v1/traces"},
		{"logs", config.OTLPForLogs(), "http://127.0.0.1:4000/v1/otlp/v1/logs"},
		{"metrics", config.OTLPForMetrics(), "http://127.0.0.1:4000/v1/otlp/v1/metrics"},
	} {
		if tc.resolved.Endpoint != tc.want {
			t.Fatalf("%s endpoint: got %q want %q", tc.signal, tc.resolved.Endpoint, tc.want)
		}
		if !isHTTPProtocol(tc.resolved.Protocol) {
			t.Fatalf("%s protocol: got %q want http", tc.signal, tc.resolved.Protocol)
		}
		if !tc.resolved.Insecure {
			t.Fatalf("%s: a http:// endpoint must export without TLS", tc.signal)
		}
	}
}

func TestGRPCProtocolWithoutEndpointUsesTheGRPCPort(t *testing.T) {
	shared := Config{OTLP: OTLPConfig{Protocol: "grpc"}}
	if got := shared.OTLPForTraces().Endpoint; got != "127.0.0.1:4317" {
		t.Fatalf("shared grpc endpoint: got %q want %q", got, "127.0.0.1:4317")
	}

	perSignal := Config{Traces: TracesConfig{OTLP: OTLPConfig{Protocol: "grpc"}}}
	if got := perSignal.OTLPForTraces().Endpoint; got != "127.0.0.1:4317" {
		t.Fatalf("signal grpc endpoint: got %q want %q", got, "127.0.0.1:4317")
	}
	if got := perSignal.OTLPForLogs().Endpoint; got != "http://127.0.0.1:4000/v1/otlp/v1/logs" {
		t.Fatalf("logs stayed on the http default: got %q", got)
	}
}

func TestSignalEndpointIsUsedVerbatim(t *testing.T) {
	config := Config{
		OTLP: OTLPConfig{Endpoint: "http://greptime:4000/v1/otlp", Protocol: "http"},
		Traces: TracesConfig{
			OTLP: OTLPConfig{Endpoint: "http://tempo:4318/v1/traces"},
		},
	}

	if got := config.OTLPForTraces().Endpoint; got != "http://tempo:4318/v1/traces" {
		t.Fatalf("signal endpoint was rewritten: %q", got)
	}
	if got := config.OTLPForLogs().Endpoint; got != "http://greptime:4000/v1/otlp/v1/logs" {
		t.Fatalf("base endpoint did not gain the logs path: %q", got)
	}
}

func TestGRPCEndpointKeepsNoSignalPathAndStaysSecure(t *testing.T) {
	config := Config{
		OTLP: OTLPConfig{Endpoint: "telemetry.example.com:1443", Protocol: "grpc"},
	}

	resolved := config.OTLPForTraces()
	if resolved.Endpoint != "telemetry.example.com:1443" {
		t.Fatalf("grpc endpoint was rewritten: %q", resolved.Endpoint)
	}
	if resolved.Insecure {
		t.Fatalf("an endpoint without a http:// scheme must keep TLS")
	}
}

func TestHTTPSEndpointKeepsTLS(t *testing.T) {
	config := Config{OTLP: OTLPConfig{Endpoint: "https://greptime.example.com/v1/otlp", Protocol: "http"}}

	resolved := config.OTLPForMetrics()
	if resolved.Insecure {
		t.Fatalf("a https:// endpoint must keep TLS")
	}
	if resolved.Endpoint != "https://greptime.example.com/v1/otlp/v1/metrics" {
		t.Fatalf("unexpected endpoint: %q", resolved.Endpoint)
	}
}

func TestMergeHeadersKeepsBaseEntriesNotRestatedBySignal(t *testing.T) {
	config := Config{
		OTLP: OTLPConfig{
			Headers: map[string]string{
				"authorization":      "Basic secret",
				"x-greptime-db-name": "public",
			},
		},
		Traces: TracesConfig{
			OTLP: OTLPConfig{
				Headers: map[string]string{"x-greptime-pipeline-name": "greptime_trace_v1"},
			},
		},
	}

	headers := config.OTLPForTraces().Headers
	want := map[string]string{
		"authorization":            "Basic secret",
		"x-greptime-db-name":       "public",
		"x-greptime-pipeline-name": "greptime_trace_v1",
	}
	if len(headers) != len(want) {
		t.Fatalf("unexpected header count: got %d want %d (%v)", len(headers), len(want), headers)
	}
	for key, value := range want {
		if headers[key] != value {
			t.Fatalf("header %q: got %q want %q", key, headers[key], value)
		}
	}
}

func TestMergeHeadersOverridesBaseCaseInsensitively(t *testing.T) {
	config := Config{
		OTLP:   OTLPConfig{Headers: map[string]string{"Authorization": "Basic base"}},
		Traces: TracesConfig{OTLP: OTLPConfig{Headers: map[string]string{"authorization": "Basic signal"}}},
	}

	headers := config.OTLPForTraces().Headers
	if len(headers) != 1 {
		t.Fatalf("expected the signal header to replace the base one, got %v", headers)
	}
	if headers["authorization"] != "Basic signal" {
		t.Fatalf("unexpected header value: %v", headers)
	}
}

func TestMergeHeadersLeavesSignalConfigUntouched(t *testing.T) {
	signalHeaders := map[string]string{"x-greptime-db-name": "public"}
	config := Config{
		OTLP:   OTLPConfig{Headers: map[string]string{"authorization": "Basic secret"}},
		Traces: TracesConfig{OTLP: OTLPConfig{Headers: signalHeaders}},
	}

	_ = config.OTLPForTraces()
	if len(signalHeaders) != 1 {
		t.Fatalf("merge mutated the signal header map: %v", signalHeaders)
	}
}

// The OTLP HTTP exporters default to TLS, so an on-prem collector reachable
// only over plain HTTP needs the insecure flag to reach the transport.
func TestHTTPTraceExporterHonoursInsecure(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		Enabled: true,
		OTLP: OTLPConfig{
			Endpoint:  server.URL + "/v1/otlp",
			Protocol:  "http",
			TimeoutMS: 2000,
		},
		Traces: TracesConfig{Exporter: "otlp"},
	}

	ctx := context.Background()
	exporter, err := NewTraceExporter(ctx, config)
	if err != nil {
		t.Fatalf("new trace exporter: %v", err)
	}

	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	_, span := provider.Tracer("otel.exporter.test").Start(ctx, "probe")
	span.End()
	if err := provider.ForceFlush(ctx); err != nil {
		t.Fatalf("force flush: %v", err)
	}
	_ = provider.Shutdown(ctx)

	if requestPath != "/v1/otlp/v1/traces" {
		t.Fatalf("collector received no export over plain http (path %q)", requestPath)
	}
}
