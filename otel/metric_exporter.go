package otel

import (
	"context"
	"strings"
	"sync/atomic"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc/credentials"
)

func httpMetricCompression(value string) otlpmetrichttp.Compression {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gzip":
		return otlpmetrichttp.GzipCompression
	default:
		return otlpmetrichttp.NoCompression
	}
}

func NewMetricExporter(ctx context.Context, config Config) (sdkmetric.Exporter, error) {
	otlpCfg := config.otlpForMetrics()
	printExporterConfig("metrics", config.Enabled, config.Metrics.Exporter, otlpCfg)
	if !config.Enabled || strings.EqualFold(strings.TrimSpace(config.Metrics.Exporter), "none") {
		return &noopMetricExporter{}, nil
	}
	if strings.HasPrefix(strings.ToLower(otlpCfg.Protocol), "http") {
		endpoint, path := otlpCfg.EndpointForHTTP()
		tlsCfg, err := otlpCfg.TLS.Load()
		if err != nil {
			return nil, err
		}
		options := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(endpoint),
			otlpmetrichttp.WithTimeout(otlpCfg.Timeout()),
			otlpmetrichttp.WithCompression(httpMetricCompression(otlpCfg.Compression)),
		}
		if path != "" {
			options = append(options, otlpmetrichttp.WithURLPath(path))
		}
		if len(otlpCfg.Headers) > 0 {
			options = append(options, otlpmetrichttp.WithHeaders(otlpCfg.Headers))
		}
		if tlsCfg != nil {
			options = append(options, otlpmetrichttp.WithTLSClientConfig(tlsCfg))
		}
		return otlpmetrichttp.New(ctx, options...)
	}

	tlsCfg, err := otlpCfg.TLS.Load()
	if err != nil {
		return nil, err
	}
	options := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(otlpCfg.EndpointForGRPC()),
		otlpmetricgrpc.WithTimeout(otlpCfg.Timeout()),
		otlpmetricgrpc.WithCompressor(otlpCfg.Compression),
	}
	if len(otlpCfg.Headers) > 0 {
		options = append(options, otlpmetricgrpc.WithHeaders(otlpCfg.Headers))
	}
	if otlpCfg.Insecure {
		options = append(options, otlpmetricgrpc.WithInsecure())
	} else if tlsCfg != nil {
		options = append(options, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
	}
	return otlpmetricgrpc.New(ctx, options...)
}

type noopMetricExporter struct {
	shutdown atomic.Bool
}

func (n *noopMetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return sdkmetric.CumulativeTemporalitySelector(kind)
}

func (n *noopMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (n *noopMetricExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	if n.shutdown.Load() {
		return sdkmetric.ErrExporterShutdown
	}
	return nil
}

func (n *noopMetricExporter) ForceFlush(context.Context) error {
	if n.shutdown.Load() {
		return sdkmetric.ErrExporterShutdown
	}
	return nil
}

func (n *noopMetricExporter) Shutdown(context.Context) error {
	n.shutdown.Store(true)
	return nil
}
