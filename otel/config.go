package otel

import (
	"net/url"
	"strings"
	"time"
)

type Config struct {
	ServiceName   string            `mapstructure:"service_name"`
	LogLevel      string            `mapstructure:"log_level" default:"info"`
	Enabled       bool              `mapstructure:"enabled" default:"false"`
	ResourceAttrs map[string]string `mapstructure:"resource_attributes"`
	OTLP          OTLPConfig        `mapstructure:"otlp"`
	Traces        TracesConfig      `mapstructure:"traces"`
	Logs          LogsConfig        `mapstructure:"logs"`
	Metrics       MetricsConfig     `mapstructure:"metrics"`
}

type OTLPConfig struct {
	Endpoint    string            `mapstructure:"endpoint" default:"http://otel-collector:4317"`
	Protocol    string            `mapstructure:"protocol" default:"grpc"`
	Headers     map[string]string `mapstructure:"headers"`
	TimeoutMS   int               `mapstructure:"timeout_ms" default:"10000"`
	Compression string            `mapstructure:"compression" default:"gzip"`
	Insecure    bool              `mapstructure:"insecure"`
	TLS         TLSConfig         `mapstructure:"tls"`
}

func (c OTLPConfig) Timeout() time.Duration {
	if c.TimeoutMS <= 0 {
		return 0
	}
	return time.Duration(c.TimeoutMS) * time.Millisecond
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

func mergeOTLP(base, override OTLPConfig) OTLPConfig {
	out := base
	if override.Endpoint != "" {
		out.Endpoint = override.Endpoint
	}
	if override.Protocol != "" {
		out.Protocol = override.Protocol
	}
	if len(override.Headers) > 0 {
		out.Headers = override.Headers
	}
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
	return mergeOTLP(c.OTLP, c.Traces.OTLP)
}

func (c Config) otlpForLogs() OTLPConfig {
	return mergeOTLP(c.OTLP, c.Logs.OTLP)
}

func (c Config) otlpForMetrics() OTLPConfig {
	return mergeOTLP(c.OTLP, c.Metrics.OTLP)
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
