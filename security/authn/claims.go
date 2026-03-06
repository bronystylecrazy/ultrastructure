package authn

import (
	"strings"

	"github.com/samber/lo"
)

func extractAPIKey(authHeader string, fallback string) string {
	authHeader = strings.TrimSpace(authHeader)
	if strings.HasPrefix(strings.ToLower(authHeader), "apikey ") {
		return strings.TrimSpace(authHeader[len("ApiKey "):])
	}
	return strings.TrimSpace(fallback)
}

func claimString(claims map[string]any, key string) string {
	v, _ := claims[key].(string)
	return v
}

func claimRoles(claims map[string]any) []string {
	out := make([]string, 0, 2)
	if role, _ := claims["role"].(string); role != "" {
		out = append(out, role)
	}
	if roles, ok := claims["roles"].([]string); ok {
		out = append(out, roles...)
		return uniqueNonEmpty(out)
	}
	if roles, ok := claims["roles"].([]any); ok {
		for _, v := range roles {
			s, _ := v.(string)
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return uniqueNonEmpty(out)
}

func claimScopes(claims map[string]any) []string {
	out := make([]string, 0, 2)
	if scope, _ := claims["scope"].(string); scope != "" {
		out = append(out, strings.Fields(scope)...)
	}
	if scopes, ok := claims["scopes"].([]string); ok {
		out = append(out, scopes...)
		return uniqueNonEmpty(out)
	}
	if scopes, ok := claims["scopes"].([]any); ok {
		for _, v := range scopes {
			s, _ := v.(string)
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return uniqueNonEmpty(out)
}

func uniqueNonEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := lo.Uniq(lo.Filter(values, func(v string, _ int) bool { return v != "" }))
	if len(out) == 0 {
		return nil
	}
	return out
}
