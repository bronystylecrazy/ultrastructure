package token

import (
	"encoding/json"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Subject   string
	TokenType string
	JTI       string
	ExpiresAt time.Time
	Values    map[string]any
}

func (c Claims) Value(key string) (any, bool) {
	v, ok := c.Values[key]
	return v, ok
}

func claimsFromJWT(in jwtgo.MapClaims) Claims {
	values := make(map[string]any, len(in))
	for k, v := range in {
		values[k] = v
	}
	return Claims{
		Subject:   claimString(values, "sub"),
		TokenType: claimString(values, "token_type"),
		JTI:       claimString(values, "jti"),
		ExpiresAt: claimUnixTime(values["exp"]),
		Values:    values,
	}
}

func claimString(values map[string]any, key string) string {
	v, _ := values[key].(string)
	return v
}

func claimUnixTime(v any) time.Time {
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
	default:
		return time.Time{}
	}
}
