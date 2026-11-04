package sqlc_test

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/ustest"
	"github.com/bronystylecrazy/ultrastructure/x/sqlc"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/fx/fxtest"
)

// pgxpool connects lazily, so none of these tests needs a server.
var postgres = database.Config{
	Driver:     "postgres",
	Datasource: "postgres://postgres:postgres@localhost:5432/postgres",
}

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

func newInstrumentedPool(t *testing.T, enabled bool, tel telemetry) *pgxpool.Pool {
	t.Helper()
	lc := fxtest.NewLifecycle(t)
	pool, err := sqlc.NewInstrumentedPool(lc, postgres, otel.Config{Enabled: enabled}, tel.tp, tel.mp)
	require.NoError(t, err)
	lc.RequireStart()
	t.Cleanup(lc.RequireStop)
	return pool
}

func TestInstrumentedPoolTracesQueriesWithoutTheirParameters(t *testing.T) {
	tel := newTelemetry()
	pool := newInstrumentedPool(t, true, tel)

	tracer, ok := pool.Config().ConnConfig.Tracer.(pgx.QueryTracer)
	require.True(t, ok, "pool has no query tracer")

	const statement = "-- name: GetUserByToken :one\nSELECT id FROM users WHERE token = $1"
	ctx, parent := tel.tp.Tracer("test").Start(context.Background(), "request")
	ctx = tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: statement, Args: []any{"s3cret-token"}})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
	parent.End()

	var query sdktrace.ReadOnlySpan
	for _, span := range tel.spans.Ended() {
		if span.Name() != "request" {
			query = span
		}
	}
	require.NotNil(t, query, "no query span was recorded")
	assert.Equal(t, "SELECT", query.Name())

	var text string
	for _, kv := range query.Attributes() {
		assert.NotContains(t, kv.Value.Emit(), "s3cret-token", "attribute %s records a query parameter", kv.Key)
		if kv.Key == "db.query.text" {
			text = kv.Value.AsString()
		}
	}
	assert.Equal(t, statement, text)
}

func TestInstrumentedPoolReportsPoolStatistics(t *testing.T) {
	tel := newTelemetry()
	newInstrumentedPool(t, true, tel)

	names := tel.metricNames(t)
	for _, want := range []string{
		"pgxpool.acquired_connections",
		"pgxpool.idle_connections",
		"pgxpool.total_connections",
		"pgxpool.max_connections",
		"pgxpool.acquires",
		"pgxpool.acquire_duration",
		"pgxpool.empty_acquire",
		"pgxpool.canceled_acquires",
	} {
		assert.Contains(t, names, want)
	}
}

func TestInstrumentedPoolAddsNothingWhenTelemetryIsDisabled(t *testing.T) {
	tel := newTelemetry()
	pool := newInstrumentedPool(t, false, tel)

	assert.Nil(t, pool.Config().ConnConfig.Tracer)
	assert.Empty(t, tel.metricNames(t))
}

// Applications get the instrumented pool from the default providers.
func TestProvidersTracePoolWhenTelemetryIsEnabled(t *testing.T) {
	var pool *pgxpool.Pool
	ustest.New(t,
		di.Replace(postgres),
		di.Replace(otel.Config{
			Enabled: true,
			Traces:  otel.TracesConfig{Exporter: "none"},
			Logs:    otel.LogsConfig{Exporter: "none"},
			Metrics: otel.MetricsConfig{Exporter: "none"},
		}),
		di.Populate(&pool),
	).RequireStart().RequireStop()

	require.NotNil(t, pool)
	assert.IsType(t, &otelpgx.Tracer{}, pool.Config().ConnConfig.Tracer)
}

func TestProvidersLeavePoolUntracedByDefault(t *testing.T) {
	var pool *pgxpool.Pool
	ustest.New(t,
		di.Replace(postgres),
		di.Populate(&pool),
	).RequireStart().RequireStop()

	require.NotNil(t, pool)
	assert.Nil(t, pool.Config().ConnConfig.Tracer)
}
