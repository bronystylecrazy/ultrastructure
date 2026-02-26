package paseto

import (
	"encoding/json"
	"time"
)

// Claims represents PASETO token claims.
type Claims struct {
	Subject   string
	TokenType string
	JTI       string
	ExpiresAt time.Time
	IssuedAt  time.Time
	NotBefore time.Time
	Issuer    string
	Values    map[string]any
}

// Value returns a claim value by key.
func (c Claims) Value(key string) (any, bool) {
	v, ok := c.Values[key]
	return v, ok
}

// fromMapClaims converts a map to Claims.
func fromMapClaims(in map[string]any) Claims {
	values := make(map[string]any, len(in))
	for k, v := range in {
		values[k] = v
	}
	return Claims{
		Subject:   claimString(values, "sub"),
		TokenType: claimString(values, "typ"),
		JTI:       claimString(values, "jti"),
		ExpiresAt: claimTime(values["exp"]),
		IssuedAt:  claimTime(values["iat"]),
		NotBefore: claimTime(values["nbf"]),
		Issuer:    claimString(values, "iss"),
		Values:    values,
	}
}

func claimString(values map[string]any, key string) string {
	v, _ := values[key].(string)
	return v
}

func claimTime(v any) time.Time {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0).UTC()
	case int64:
		return time.Unix(t, 0).UTC()
	case int:
		return time.Unix(int64(t), 0).UTC()
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return time.Time{}
		}
		return time.Unix(i, 0).UTC()
	case string:
		parsed, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return time.Time{}
		}
		return parsed.UTC()
	default:
		return time.Time{}
	}
}
