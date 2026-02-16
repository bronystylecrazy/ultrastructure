package otel

import (
	"strings"
)

func printExporterConfig(signal string, enabled bool, exporter string, cfg OTLPConfig) {
	// fmt.Println(
	// 	"[otel]",
	// 	"signal="+signal,
	// 	"enabled=", enabled,
	// 	"exporter="+strings.TrimSpace(exporter),
	// 	"protocol="+cfg.Protocol,
	// 	"endpoint="+cfg.Endpoint,
	// 	"timeout_ms=", cfg.TimeoutMS,
	// 	"compression="+cfg.Compression,
	// 	"insecure=", cfg.Insecure,
	// 	"headers=", maskHeaders(cfg.Headers),
	// )
}

func maskHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		lk := strings.ToLower(strings.TrimSpace(k))
		if strings.Contains(lk, "authorization") || strings.Contains(lk, "token") || strings.Contains(lk, "secret") || strings.Contains(lk, "key") {
			out[k] = maskValue(v)
			continue
		}
		out[k] = v
	}
	return out
}

func maskValue(v string) string {
	r := []rune(strings.TrimSpace(v))
	n := len(r)
	if n <= 4 {
		return "****"
	}
	return string(r[:2]) + "****" + string(r[n-2:])
}
