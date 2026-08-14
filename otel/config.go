package otel

import (
	"net/url"
	"strings"
	"time"
)

type Config struct {
	ServiceName   string            `mapstructure:"service_name"`
	LogLevel      string            `mapstructure:"log_level" default:"info"`
	LogAllowlist  []string          `mapstructure:"log_allowlist"`
	Enabled       bool              `mapstructure:"enabled" default:"false"`
	ResourceAttrs map[string]string `mapstructure:"resource_attributes"`
	OTLP          OTLPConfig        `mapstructure:"otlp"`
	Traces        TracesConfig      `mapstructure:"traces"`
	Logs          LogsConfig        `mapstructure:"logs"`
	Metrics       MetricsConfig     `mapstructure:"metrics"`
}

func (c Config) effectiveLogAllowlist() []string {
	return append([]string{}, c.LogAllowlist...)
}

type OTLPConfig struct {
	Endpoint    string            `mapstructure:"endpoint"`
	Protocol    string            `mapstructure:"protocol"`
	Headers     map[string]string `mapstructure:"headers"`
	TimeoutMS   int               `mapstructure:"timeout_ms"`
	Compression string            `mapstructure:"compression"`
	Insecure    bool              `mapstructure:"insecure"`
	TLS         TLSConfig         `mapstructure:"tls"`
}

const (
	// A local GreptimeDB standalone is the default collector: it serves OTLP
	// over HTTP only, on port 4000 under /v1/otlp, and appends the per-signal
	// path the OTLP specification defines.
	defaultOTLPHTTPEndpoint = "http://127.0.0.1:4000/v1/otlp"
	// Collectors that speak OTLP over gRPC listen on the standard port instead.
	defaultOTLPGRPCEndpoint = "127.0.0.1:4317"
	defaultOTLPProtocol     = "http"
	defaultOTLPTimeoutMS    = 10000
	defaultOTLPCompression  = "gzip"
)

// defaultOTLPEndpointFor returns the local collector endpoint for a protocol,
// so that choosing gRPC does not aim the exporter at the HTTP port.
func defaultOTLPEndpointFor(protocol string) string {
	if isHTTPProtocol(protocol) {
		return defaultOTLPHTTPEndpoint
	}
	return defaultOTLPGRPCEndpoint
}

// Per-signal paths appended to an OTLP/HTTP base endpoint, as defined by the
// OTLP specification. A signal that carries its own endpoint is used verbatim.
const (
	tracesSignalPath  = "/v1/traces"
	logsSignalPath    = "/v1/logs"
	metricsSignalPath = "/v1/metrics"
)

func (c OTLPConfig) Timeout() time.Duration {
	if c.TimeoutMS <= 0 {
		return 0
	}
	return time.Duration(c.TimeoutMS) * time.Millisecond
}

func (c OTLPConfig) withDefaults() OTLPConfig {
	out := c
	if out.Protocol == "" {
		out.Protocol = defaultOTLPProtocol
	}
	if out.Endpoint == "" {
		out.Endpoint = defaultOTLPEndpointFor(out.Protocol)
	}
	if out.TimeoutMS <= 0 {
		out.TimeoutMS = defaultOTLPTimeoutMS
	}
	if out.Compression == "" {
		out.Compression = defaultOTLPCompression
	}
	return out
}

func (c OTLPConfig) EndpointForGRPC() string {
	endpoint := strings.TrimSpace(c.Endpoint)
	if endpoint == "" {
		return endpoint
	}
	if !strings.Contains(endpoint, "://") {
		return endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	if u.Host != "" {
		return u.Host
	}
	return endpoint
}

func (c OTLPConfig) EndpointForHTTP() (string, string) {
	endpoint := strings.TrimSpace(c.Endpoint)
	if endpoint == "" {
		return "", ""
	}
	if strings.Contains(endpoint, "://") {
		u, err := url.Parse(endpoint)
		if err != nil {
			return endpoint, ""
		}
		host := u.Host
		path := strings.TrimSpace(u.Path)
		return host, path
	}
	if strings.Contains(endpoint, "/") {
		u, err := url.Parse("http://" + endpoint)
		if err != nil {
			return endpoint, ""
		}
		host := u.Host
		path := strings.TrimSpace(u.Path)
		return host, path
	}
	return endpoint, ""
}

type TracesConfig struct {
	Exporter   string     `mapstructure:"exporter" default:"none"`
	Sampler    string     `mapstructure:"sampler" default:"parentbased_traceidratio"`
	SamplerArg float64    `mapstructure:"sampler_arg" default:"1"`
	OTLP       OTLPConfig `mapstructure:"otlp"`
}

type LogsConfig struct {
	Exporter string     `mapstructure:"exporter" default:"none"`
	OTLP     OTLPConfig `mapstructure:"otlp"`
}

type MetricsConfig struct {
	Exporter string        `mapstructure:"exporter" default:"none"`
	Tuning   MetricsTuning `mapstructure:"tuning"`
	OTLP     OTLPConfig    `mapstructure:"otlp"`
}

type MetricsTuning struct {
	ExportIntervalMS     int    `mapstructure:"export_interval_ms" default:"10000"`
	Temporality          string `mapstructure:"temporality" default:"cumulative"`
	HistogramAggregation string `mapstructure:"histogram_aggregation" default:"explicit_bucket_histogram"`
}

// mergeHeaders combines the shared OTLP headers with a signal's own headers,
// keeping every base entry the signal does not restate. Names are matched
// case-insensitively so a signal can override a base header regardless of the
// casing each config uses, without emitting the header twice.
func mergeHeaders(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}

	out := make(map[string]string, len(base)+len(override))
	keysByName := make(map[string]string, len(base)+len(override))
	put := func(key, value string) {
		name := strings.ToLower(strings.TrimSpace(key))
		if previous, ok := keysByName[name]; ok {
			delete(out, previous)
		}
		keysByName[name] = key
		out[key] = value
	}

	for key, value := range base {
		put(key, value)
	}
	for key, value := range override {
		put(key, value)
	}
	return out
}

func mergeOTLP(base, override OTLPConfig) OTLPConfig {
	out := base
	if override.Endpoint != "" {
		out.Endpoint = override.Endpoint
	}
	if override.Protocol != "" {
		out.Protocol = override.Protocol
	}
	out.Headers = mergeHeaders(base.Headers, override.Headers)
	if override.TimeoutMS > 0 {
		out.TimeoutMS = override.TimeoutMS
	}
	if override.Compression != "" {
		out.Compression = override.Compression
	}
	if override.Insecure {
		out.Insecure = true
	}
	out.TLS = mergeTLS(base.TLS, override.TLS)
	return out
}

func (c Config) otlpForTraces() OTLPConfig {
	return resolveSignalOTLP(c.OTLP, c.Traces.OTLP, tracesSignalPath)
}

func (c Config) otlpForLogs() OTLPConfig {
	return resolveSignalOTLP(c.OTLP, c.Logs.OTLP, logsSignalPath)
}

func (c Config) otlpForMetrics() OTLPConfig {
	return resolveSignalOTLP(c.OTLP, c.Metrics.OTLP, metricsSignalPath)
}

// resolveSignalOTLP merges the shared OTLP settings with one signal's own, then
// completes the endpoint: an OTLP/HTTP base endpoint gains the signal path, and
// an endpoint written as http:// is exported without TLS.
func resolveSignalOTLP(base, override OTLPConfig, signalPath string) OTLPConfig {
	// Defaults are applied after the merge so that a signal choosing gRPC gets
	// the gRPC endpoint rather than the shared block's HTTP one.
	out := mergeOTLP(base, override).withDefaults()
	if strings.TrimSpace(override.Endpoint) == "" && isHTTPProtocol(out.Protocol) {
		out.Endpoint = strings.TrimRight(strings.TrimSpace(out.Endpoint), "/") + signalPath
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(out.Endpoint)), "http://") {
		out.Insecure = true
	}
	return out
}

func isHTTPProtocol(protocol string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(protocol)), "http")
}

// OTLPForTraces returns the merged OTLP config for traces.
func (c Config) OTLPForTraces() OTLPConfig {
	return c.otlpForTraces()
}

// OTLPForLogs returns the merged OTLP config for logs.
func (c Config) OTLPForLogs() OTLPConfig {
	return c.otlpForLogs()
}

// OTLPForMetrics returns the merged OTLP config for metrics.
func (c Config) OTLPForMetrics() OTLPConfig {
	return c.otlpForMetrics()
}
