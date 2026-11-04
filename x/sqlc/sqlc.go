package sqlc

import (
	"context"

	"github.com/bronystylecrazy/ultrastructure/database"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/fx"
)

// NewPool builds the pool without telemetry; NewInstrumentedPool is what
// Providers registers.
func NewPool(lc fx.Lifecycle, config database.Config) (*pgxpool.Pool, error) {
	return NewInstrumentedPool(lc, config, otel.Config{}, nil, nil)
}

// NewInstrumentedPool builds the pool and, when telemetry is enabled, traces
// its queries through tp and reports its statistics through mp. Query spans
// carry the statement text but never the values bound to it. With telemetry
// disabled it builds the same pool as NewPool.
func NewInstrumentedPool(lc fx.Lifecycle, config database.Config, otelConfig otel.Config, tp *otel.TracerProvider, mp *otel.MeterProvider) (*pgxpool.Pool, error) {
	if database.ParseDialect(config.Driver) != "postgres" {
		return nil, nil
	}

	poolConfig, err := pgxpool.ParseConfig(config.Datasource)
	if err != nil {
		return nil, err
	}
	if otelConfig.Enabled && tp != nil {
		poolConfig.ConnConfig.Tracer = newQueryTracer(tp, mp)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	if otelConfig.Enabled && mp != nil {
		if err := otelpgx.RecordStats(pool, otelpgx.WithStatsMeterProvider(mp)); err != nil {
			pool.Close()
			return nil, err
		}
	}

	lc.Append(
		fx.Hook{
			OnStop: func(ctx context.Context) error {
				pool.Close()
				return nil
			},
		},
	)

	return pool, nil
}

// newQueryTracer leaves query parameters out of spans, which is otelpgx's
// default; do not add otelpgx.WithIncludeQueryParameters.
func newQueryTracer(tp *otel.TracerProvider, mp *otel.MeterProvider) *otelpgx.Tracer {
	opts := []otelpgx.Option{otelpgx.WithTracerProvider(tp)}
	if mp != nil {
		opts = append(opts, otelpgx.WithMeterProvider(mp))
	} else {
		opts = append(opts, otelpgx.WithMeterProvider(noop.NewMeterProvider()))
	}
	return otelpgx.NewTracer(opts...)
}
