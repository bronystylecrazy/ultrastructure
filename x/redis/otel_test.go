package rd_test

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	rd "github.com/bronystylecrazy/ultrastructure/x/redis"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type telemetry struct {
	spans  *tracetest.SpanRecorder
	reader *sdkmetric.ManualReader
	tp     *otel.TracerProvider
	mp     *otel.MeterProvider
}

func newTelemetry() telemetry {
	spans := tracetest.NewSpanRecorder()
	reader := sdkmetric.NewManualReader()
	return telemetry{
		spans:  spans,
		reader: reader,
		tp:     &otel.TracerProvider{TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))},
		mp:     &otel.MeterProvider{MeterProvider: sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))},
	}
}

func (tel telemetry) metricNames(t *testing.T) []string {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, tel.reader.Collect(context.Background(), &rm))
	var names []string
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			names = append(names, m.Name)
		}
	}
	return names
}

func newInstrumentedClient(t *testing.T, enabled bool, tel telemetry) *redis.Client {
	t.Helper()
	client, err := rd.NewInstrumentedClient(rd.Config{InMemory: true}, otel.Config{Enabled: enabled}, tel.tp, tel.mp)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestInstrumentedClientTracesCommandsWithoutTheirArguments(t *testing.T) {
	tel := newTelemetry()
	client := newInstrumentedClient(t, true, tel)

	require.NoError(t, client.Set(context.Background(), "session:abc", "s3cret-value", 0).Err())

	var set sdktrace.ReadOnlySpan
	for _, span := range tel.spans.Ended() {
		if span.Name() == "set" {
			set = span
		}
	}
	require.NotNil(t, set, "no span was recorded for SET")
	for _, kv := range set.Attributes() {
		assert.NotContains(t, kv.Value.Emit(), "s3cret-value", "attribute %s records a command argument", kv.Key)
	}
}

func TestInstrumentedClientReportsPoolAndCommandMetrics(t *testing.T) {
	tel := newTelemetry()
	client := newInstrumentedClient(t, true, tel)

	require.NoError(t, client.Ping(context.Background()).Err())

	names := tel.metricNames(t)
	for _, want := range []string{
		"db.client.connections.usage",
		"db.client.connections.max",
		"db.client.connections.timeouts",
		"db.client.connections.use_time",
	} {
		assert.Contains(t, names, want)
	}
}

func TestInstrumentedClientAddsNothingWhenTelemetryIsDisabled(t *testing.T) {
	tel := newTelemetry()
	client := newInstrumentedClient(t, false, tel)

	require.NoError(t, client.Ping(context.Background()).Err())

	assert.Empty(t, tel.spans.Ended())
	assert.Empty(t, tel.metricNames(t))
}

// Applications get the instrumented client from Providers.
func TestProvidersTraceClientWhenTelemetryIsEnabled(t *testing.T) {
	tel := newTelemetry()
	var client *redis.Client
	app := ustest.New(t,
		rd.Providers(),
		di.Replace(rd.Config{InMemory: true}),
		di.Replace(otel.Config{
			Enabled: true,
			Traces:  otel.TracesConfig{Exporter: "none"},
			Logs:    otel.LogsConfig{Exporter: "none"},
			Metrics: otel.MetricsConfig{Exporter: "none"},
		}),
		di.Replace(tel.tp),
		di.Populate(&client),
	).RequireStart()
	defer app.RequireStop()

	require.NotNil(t, client)
	require.NoError(t, client.Ping(context.Background()).Err())
	assert.NotEmpty(t, tel.spans.Ended())
}
