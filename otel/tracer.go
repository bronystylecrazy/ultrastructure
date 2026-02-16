package otel

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
)

func httpTraceCompression(value string) otlptracehttp.Compression {

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gzip":
		return otlptracehttp.GzipCompression
	default:
		return otlptracehttp.NoCompression
	}
}

func NewTraceExporter(ctx context.Context, config Config, opts ...otlptracegrpc.Option) (sdktrace.SpanExporter, error) {
	otlpCfg := config.otlpForTraces()
	printExporterConfig("traces", config.Enabled, config.Traces.Exporter, otlpCfg)
	if !config.Enabled || strings.EqualFold(strings.TrimSpace(config.Traces.Exporter), "none") {
		return nil, nil
	}
	if strings.HasPrefix(strings.ToLower(otlpCfg.Protocol), "http") {
		endpoint, path := otlpCfg.EndpointForHTTP()
		tlsCfg, err := otlpCfg.TLS.Load()
		if err != nil {
			return nil, err
		}
		options := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithTimeout(otlpCfg.Timeout()),
			otlptracehttp.WithCompression(httpTraceCompression(otlpCfg.Compression)),
		}
		if path != "" {
			options = append(options, otlptracehttp.WithURLPath(path))
		}
		if len(otlpCfg.Headers) > 0 {
			options = append(options, otlptracehttp.WithHeaders(otlpCfg.Headers))
		}
		if tlsCfg != nil {
			options = append(options, otlptracehttp.WithTLSClientConfig(tlsCfg))
		}
		return otlptracehttp.New(ctx, options...)
	}

	tlsCfg, err := otlpCfg.TLS.Load()
	if err != nil {
		return nil, err
	}
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(otlpCfg.EndpointForGRPC()),
		otlptracegrpc.WithTimeout(otlpCfg.Timeout()),
		otlptracegrpc.WithCompressor(otlpCfg.Compression),
	}
	if len(otlpCfg.Headers) > 0 {
		options = append(options, otlptracegrpc.WithHeaders(otlpCfg.Headers))
	}
	if otlpCfg.Insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	} else if tlsCfg != nil {
		options = append(options, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
	}

	for _, option := range opts {
		options = append(options, option)
	}

	return otlptracegrpc.New(ctx, options...)
}
