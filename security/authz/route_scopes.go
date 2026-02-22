package authz

import (
	"strings"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

type RouteScopeOption func(*routeScopeConfig)

type routeScopeConfig struct {
	registry    *web.MetadataRegistry
	userSchemes map[string]struct{}
	appSchemes  map[string]struct{}
}

func defaultRouteScopeConfig() routeScopeConfig {
	return routeScopeConfig{
		registry: nil,
		userSchemes: toSet(
			"BearerAuth",
			"OAuth2",
			"UserToken",
			"JWT",
		),
		appSchemes: toSet(
			"ApiKeyAuth",
			"ApiKey",
			"APIKey",
		),
	}
}

func WithScopeRegistry(registry *web.MetadataRegistry) RouteScopeOption {
	return func(c *routeScopeConfig) {
		if registry != nil {
			c.registry = registry
		}
	}
}

func WithUserScopeSchemes(schemes ...string) RouteScopeOption {
	return func(c *routeScopeConfig) {
		if len(schemes) > 0 {
			c.userSchemes = toSet(schemes...)
		}
	}
}

func WithAppScopeSchemes(schemes ...string) RouteScopeOption {
	return func(c *routeScopeConfig) {
		if len(schemes) > 0 {
			c.appSchemes = toSet(schemes...)
		}
	}
}

func RequireRouteScopes(opts ...RouteScopeOption) fiber.Handler {
	cfg := defaultRouteScopeConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil {
			return denyUnauthorized(c)
		}
		if isSuperAdmin(p) {
			return c.Next()
		}
		if !enforceRouteScopes(c, p, cfg) {
			return nil
		}
		return c.Next()
	}
}

func enforceRouteScopes(c fiber.Ctx, p *authn.Principal, cfg routeScopeConfig) bool {
	meta := lookupRouteMetadata(cfg.registry, c)
	if meta == nil || len(meta.Security) == 0 {
		return true
	}

	requirements := relevantSecurityRequirements(meta.Security, p.Type, cfg)
	if len(requirements) == 0 {
		_ = denyForbidden(c)
		return false
	}
	for _, req := range requirements {
		if len(req.Scopes) == 0 {
			return true
		}
		if hasAllScopes(p.Scopes, req.Scopes...) {
			return true
		}
	}
	_ = denyForbidden(c)
	return false
}

func lookupRouteMetadata(registry *web.MetadataRegistry, c fiber.Ctx) *web.RouteMetadata {
	if registry == nil {
		return nil
	}
	method := c.Method()
	candidates := make([]string, 0, 4)

	if route := c.Route(); route != nil {
		if rp := strings.TrimSpace(route.Path); rp != "" {
			candidates = append(candidates, rp)
		}
	}
	if cp := strings.TrimSpace(c.Path()); cp != "" {
		candidates = append(candidates, cp)
	}

	seen := map[string]struct{}{}
	for _, raw := range candidates {
		for _, p := range normalizeLookupPaths(raw) {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			if meta := registry.GetRoute(method, p); meta != nil {
				return meta
			}
		}
	}
	return nil
}

func normalizeLookupPaths(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if path == "/" {
		return []string{"/"}
	}
	out := []string{path}
	trimmed := strings.TrimSuffix(path, "/")
	if trimmed != path {
		out = append(out, trimmed)
	}
	if !strings.HasPrefix(path, "/") {
		out = append(out, "/"+path)
	}
	return out
}

func relevantSecurityRequirements(in []web.SecurityRequirement, principalType authn.PrincipalType, cfg routeScopeConfig) []web.SecurityRequirement {
	if len(in) == 0 {
		return nil
	}
	out := make([]web.SecurityRequirement, 0, len(in))
	for _, req := range in {
		scheme := strings.TrimSpace(req.Scheme)
		if scheme == "" {
			continue
		}
		switch principalType {
		case authn.PrincipalUser:
			if _, ok := cfg.userSchemes[scheme]; ok {
				out = append(out, req)
			}
		case authn.PrincipalApp:
			if _, ok := cfg.appSchemes[scheme]; ok {
				out = append(out, req)
			}
		}
	}
	return out
}

func toSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}
