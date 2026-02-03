package otel

import "time"

type Config struct {
	Env        string        `mapstructure:"env"`
	Level      string        `mapstructure:"level"`
	Disabled   bool          `mapstructure:"disabled"`
	Compressor string        `mapstructure:"compressor"` // or OTEL_EXPORTER_OTLP_COMPRESSION
	Endpoint   string        `mapstructure:"endpoint"`   // or OTEL_EXPORTER_OTLP_ENDPOINT
	AuthKey    string        `mapstructure:"auth_key"`
	Service    string        `mapstructure:"service"`
	Namespace  string        `mapstructure:"namespace"`
	Timeout    time.Duration `mapstructure:"timeout"` // or OTEL_EXPORTER_OTLP_TIMEOUT
}
