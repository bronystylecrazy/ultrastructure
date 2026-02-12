package otel

import (
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

func DefaultMetricAttributes(config Config) []attribute.KeyValue {
	environment := ""
	if len(config.ResourceAttrs) > 0 {
		environment = strings.TrimSpace(config.ResourceAttrs["deployment.environment"])
	}
	if environment == "" {
		return nil
	}
	return []attribute.KeyValue{
		attribute.String("deployment.environment", environment),
	}
}

func MergeMetricAttributes(defaults []attribute.KeyValue, attrs []attribute.KeyValue) []attribute.KeyValue {
	if len(defaults) == 0 {
		return attrs
	}
	if len(attrs) == 0 {
		return defaults
	}

	keys := make(map[attribute.Key]struct{}, len(attrs))
	for _, kv := range attrs {
		keys[kv.Key] = struct{}{}
	}

	out := make([]attribute.KeyValue, 0, len(defaults)+len(attrs))
	for _, kv := range defaults {
		if _, ok := keys[kv.Key]; ok {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, attrs...)
	return out
}
