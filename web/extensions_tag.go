package web

import (
	"strconv"
	"strings"
)

func applyExtensionsTag(schema map[string]interface{}, raw string) {
	if schema == nil {
		return
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}

	for _, token := range strings.Split(raw, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		negated := strings.HasPrefix(token, "!")
		if negated {
			token = strings.TrimSpace(strings.TrimPrefix(token, "!"))
			if !strings.HasPrefix(token, "x-") {
				continue
			}
			schema[token] = false
			continue
		}

		if strings.Contains(token, "=") {
			parts := strings.SplitN(token, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if !strings.HasPrefix(key, "x-") {
				continue
			}
			schema[key] = parseExtensionValue(value)
			continue
		}

		if !strings.HasPrefix(token, "x-") {
			continue
		}
		schema[token] = true
	}
}

func parseExtensionValue(raw string) interface{} {
	if b, err := strconv.ParseBool(raw); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	return raw
}
