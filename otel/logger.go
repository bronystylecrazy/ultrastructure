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

// provideLogExporter is NewLogExporter's DI registration shape: the variadic
// options are a direct-call convenience only. Registering the variadic
// function itself breaks the graph — the auto-group wrap goes through
// fx.Annotate, which turns a variadic parameter into a REQUIRED
// []otlploggrpc.Option dependency nobody provides — and since the exporters
// joined the lc.stoppers group (built at start), every consumer app failed
// with "missing type: []otlploggrpc.Option".
func provideLogExporter(ctx context.Context, config Config) (sdklog.Exporter, error) {
	return NewLogExporter(ctx, config)
}

func NewLogExporter(ctx context.Context, config Config, opts ...otlploggrpc.Option) (sdklog.Exporter, error) {
	if !config.Enabled || strings.EqualFold(strings.TrimSpace(config.Logs.Exporter), "none") {
		return nil, nil
	}
	otlpCfg := config.otlpForLogs()
	if isHTTPProtocol(otlpCfg.Protocol) {
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
		if otlpCfg.Insecure {
			options = append(options, otlploghttp.WithInsecure())
		} else if tlsCfg != nil {
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
