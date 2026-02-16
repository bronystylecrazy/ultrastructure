package otel

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc/credentials"
)

func httpLogCompression(value string) otlploghttp.Compression {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gzip":
		return otlploghttp.GzipCompression
	default:
		return otlploghttp.NoCompression
	}
}

func NewLogExporter(ctx context.Context, config Config, opts ...otlploggrpc.Option) (sdklog.Exporter, error) {
	otlpCfg := config.otlpForLogs()
	printExporterConfig("logs", config.Enabled, config.Logs.Exporter, otlpCfg)
	if !config.Enabled || strings.EqualFold(strings.TrimSpace(config.Logs.Exporter), "none") {
		return nil, nil
	}
	if strings.HasPrefix(strings.ToLower(otlpCfg.Protocol), "http") {
		endpoint, path := otlpCfg.EndpointForHTTP()
		tlsCfg, err := otlpCfg.TLS.Load()
		if err != nil {
			return nil, err
		}
		options := []otlploghttp.Option{
			otlploghttp.WithEndpoint(endpoint),
			otlploghttp.WithTimeout(otlpCfg.Timeout()),
			otlploghttp.WithCompression(httpLogCompression(otlpCfg.Compression)),
		}
		if path != "" {
			options = append(options, otlploghttp.WithURLPath(path))
		}
		if len(otlpCfg.Headers) > 0 {
			options = append(options, otlploghttp.WithHeaders(otlpCfg.Headers))
		}
		if tlsCfg != nil {
			options = append(options, otlploghttp.WithTLSClientConfig(tlsCfg))
		}
		return otlploghttp.New(ctx, options...)
	}

	tlsCfg, err := otlpCfg.TLS.Load()
	if err != nil {
		return nil, err
	}
	options := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(otlpCfg.EndpointForGRPC()),
		otlploggrpc.WithTimeout(otlpCfg.Timeout()),
		otlploggrpc.WithCompressor(otlpCfg.Compression),
	}

	if len(otlpCfg.Headers) > 0 {
		options = append(options, otlploggrpc.WithHeaders(otlpCfg.Headers))
	}
	if otlpCfg.Insecure {
		options = append(options, otlploggrpc.WithInsecure())
	} else if tlsCfg != nil {
		options = append(options, otlploggrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
	}

	for _, option := range opts {
		options = append(options, option)
	}

	return otlploggrpc.New(ctx, options...)
}
