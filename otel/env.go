package otel

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func applyOTELenv(v *viper.Viper) error {
	if v == nil {
		return nil
	}
	applyStringEnv(v, "otel.service_name", "OTEL_SERVICE_NAME")
	applyBoolEnv(v, "otel.enabled", "OTEL_ENABLED")
	applyStringEnv(v, "otel.traces.exporter", "OTEL_TRACES_EXPORTER")
	applyStringEnv(v, "otel.logs.exporter", "OTEL_LOGS_EXPORTER")
	applyStringEnv(v, "otel.metrics.exporter", "OTEL_METRICS_EXPORTER")
	applyResourceAttrsEnv(v, "otel.resource_attributes", "OTEL_RESOURCE_ATTRIBUTES")

	applyStringEnv(v, "otel.otlp.endpoint", "OTEL_EXPORTER_OTLP_ENDPOINT")
	applyStringEnv(v, "otel.otlp.protocol", "OTEL_EXPORTER_OTLP_PROTOCOL")
	applyStringEnv(v, "otel.otlp.compression", "OTEL_EXPORTER_OTLP_COMPRESSION")
	applyTimeoutEnv(v, "otel.otlp.timeout_ms", "OTEL_EXPORTER_OTLP_TIMEOUT")
	applyHeadersEnv(v, "otel.otlp.headers", "OTEL_EXPORTER_OTLP_HEADERS")
	applyBoolEnv(v, "otel.otlp.insecure", "OTEL_EXPORTER_OTLP_INSECURE")
	applyStringEnv(v, "otel.otlp.tls.ca_file", "OTEL_EXPORTER_OTLP_CERTIFICATE")
	applyStringEnv(v, "otel.otlp.tls.cert_file", "OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE")
	applyStringEnv(v, "otel.otlp.tls.key_file", "OTEL_EXPORTER_OTLP_CLIENT_KEY")

	applyStringEnv(v, "otel.traces.otlp.endpoint", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	applyStringEnv(v, "otel.traces.otlp.protocol", "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL")
	applyStringEnv(v, "otel.traces.otlp.compression", "OTEL_EXPORTER_OTLP_TRACES_COMPRESSION")
	applyTimeoutEnv(v, "otel.traces.otlp.timeout_ms", "OTEL_EXPORTER_OTLP_TRACES_TIMEOUT")
	applyHeadersEnv(v, "otel.traces.otlp.headers", "OTEL_EXPORTER_OTLP_TRACES_HEADERS")
	applyBoolEnv(v, "otel.traces.otlp.insecure", "OTEL_EXPORTER_OTLP_TRACES_INSECURE")
	applyStringEnv(v, "otel.traces.otlp.tls.ca_file", "OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE")
	applyStringEnv(v, "otel.traces.otlp.tls.cert_file", "OTEL_EXPORTER_OTLP_TRACES_CLIENT_CERTIFICATE")
	applyStringEnv(v, "otel.traces.otlp.tls.key_file", "OTEL_EXPORTER_OTLP_TRACES_CLIENT_KEY")

	applyStringEnv(v, "otel.logs.otlp.endpoint", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
	applyStringEnv(v, "otel.logs.otlp.protocol", "OTEL_EXPORTER_OTLP_LOGS_PROTOCOL")
	applyStringEnv(v, "otel.logs.otlp.compression", "OTEL_EXPORTER_OTLP_LOGS_COMPRESSION")
	applyTimeoutEnv(v, "otel.logs.otlp.timeout_ms", "OTEL_EXPORTER_OTLP_LOGS_TIMEOUT")
	applyHeadersEnv(v, "otel.logs.otlp.headers", "OTEL_EXPORTER_OTLP_LOGS_HEADERS")
	applyBoolEnv(v, "otel.logs.otlp.insecure", "OTEL_EXPORTER_OTLP_LOGS_INSECURE")
	applyStringEnv(v, "otel.logs.otlp.tls.ca_file", "OTEL_EXPORTER_OTLP_LOGS_CERTIFICATE")
	applyStringEnv(v, "otel.logs.otlp.tls.cert_file", "OTEL_EXPORTER_OTLP_LOGS_CLIENT_CERTIFICATE")
	applyStringEnv(v, "otel.logs.otlp.tls.key_file", "OTEL_EXPORTER_OTLP_LOGS_CLIENT_KEY")

	applyStringEnv(v, "otel.metrics.otlp.endpoint", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
	applyStringEnv(v, "otel.metrics.otlp.protocol", "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL")
	applyStringEnv(v, "otel.metrics.otlp.compression", "OTEL_EXPORTER_OTLP_METRICS_COMPRESSION")
	applyTimeoutEnv(v, "otel.metrics.otlp.timeout_ms", "OTEL_EXPORTER_OTLP_METRICS_TIMEOUT")
	applyHeadersEnv(v, "otel.metrics.otlp.headers", "OTEL_EXPORTER_OTLP_METRICS_HEADERS")
	applyBoolEnv(v, "otel.metrics.otlp.insecure", "OTEL_EXPORTER_OTLP_METRICS_INSECURE")
	applyStringEnv(v, "otel.metrics.otlp.tls.ca_file", "OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE")
	applyStringEnv(v, "otel.metrics.otlp.tls.cert_file", "OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE")
	applyStringEnv(v, "otel.metrics.otlp.tls.key_file", "OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY")

	applyStringEnv(v, "otel.traces.sampler", "OTEL_TRACES_SAMPLER")
	applyFloatEnv(v, "otel.traces.sampler_arg", "OTEL_TRACES_SAMPLER_ARG")

	return nil
}

func applyStringEnv(v *viper.Viper, key string, env string) {
	if val, ok := os.LookupEnv(env); ok {
		v.Set(key, strings.TrimSpace(val))
	}
}

func applyTimeoutEnv(v *viper.Viper, key string, env string) {
	if val, ok := os.LookupEnv(env); ok {
		if ms, ok := ParseTimeoutMS(val); ok {
			v.Set(key, ms)
		}
	}
}

func applyFloatEnv(v *viper.Viper, key string, env string) {
	if val, ok := os.LookupEnv(env); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			v.Set(key, f)
		}
	}
}

func applyBoolEnv(v *viper.Viper, key string, env string) {
	if val, ok := os.LookupEnv(env); ok {
		if b, err := strconv.ParseBool(strings.TrimSpace(val)); err == nil {
			v.Set(key, b)
		}
	}
}

func applyHeadersEnv(v *viper.Viper, key string, env string) {
	if val, ok := os.LookupEnv(env); ok {
		if headers := ParseHeaders(val); len(headers) > 0 {
			v.Set(key, headers)
		}
	}
}

func applyResourceAttrsEnv(v *viper.Viper, key string, env string) {
	if v == nil {
		return
	}
	val, ok := os.LookupEnv(env)
	if !ok {
		return
	}
	attrs := ParseHeaders(val)
	if len(attrs) == 0 {
		return
	}
	existing := map[string]string{}
	if current := v.GetStringMapString(key); len(current) > 0 {
		for k, v := range current {
			existing[k] = v
		}
	}
	for k, v := range attrs {
		existing[k] = v
	}
	v.Set(key, existing)
}

func ParseTimeoutMS(value string) (int, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(trimmed); err == nil {
		return n, true
	}
	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, false
	}
	return int(d / time.Millisecond), true
}

func ParseHeaders(value string) map[string]string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make(map[string]string)
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
