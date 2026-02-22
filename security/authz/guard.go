package authz

import (
	"sort"
	"strings"
	"sync"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	httpx "github.com/bronystylecrazy/ultrastructure/security/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

type ConflictPolicy string

const (
	PolicyPreferUser     ConflictPolicy = "prefer_user"
	PolicyPreferApp      ConflictPolicy = "prefer_app"
	PolicyDenyIfMultiple ConflictPolicy = "deny_if_multiple"
	SuperAdminRole       string         = "super_admin"
)

var (
	superAdminRolesMu  sync.RWMutex
	superAdminRolesSet = map[string]struct{}{SuperAdminRole: {}}
)

func RequireAnyPrincipal() fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, ok := authn.PrincipalFromContext(c.Context()); !ok {
			return denyUnauthorized(c)
		}
		return c.Next()
	}
}

func ResolvePolicy(policy ConflictPolicy, opts ...RouteScopeOption) fiber.Handler {
	cfg := defaultRouteScopeConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return func(c fiber.Ctx) error {
		p, err := principalByPolicy(c, policy)
		if err != nil {
			switch err {
			case errUnauthorized:
				return denyUnauthorized(c)
			default:
				return denyForbidden(c)
			}
		}
		c.SetContext(authn.WithPrincipal(c.Context(), p))
		authn.SetPrincipalLocals(c, p)
		if isSuperAdmin(p) {
			return c.Next()
		}
		if !enforceRouteScopes(c, p, cfg) {
			return nil
		}
		return c.Next()
	}
}

func RequireUserRole(role string) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil || p.Type != authn.PrincipalUser {
			return denyForbidden(c)
		}
		for _, r := range p.Roles {
			if r == role {
				return c.Next()
			}
		}
		return denyForbidden(c)
	}
}

var (
	errUnauthorized = fiber.ErrUnauthorized
	errConflict     = fiber.ErrForbidden
)

func principalByPolicy(c fiber.Ctx, policy ConflictPolicy) (*authn.Principal, error) {
	principals, ok := authn.PrincipalsFromContext(c.Context())
	if !ok || len(principals) == 0 {
		if p, ok := authn.PrincipalFromContext(c.Context()); ok && p != nil {
			return p, nil
		}
		return nil, errUnauthorized
	}
	if len(principals) == 1 {
		return principals[0], nil
	}

	switch policy {
	case PolicyDenyIfMultiple:
		return nil, errConflict
	case PolicyPreferApp:
		for _, p := range principals {
			if p != nil && p.Type == authn.PrincipalApp {
				return p, nil
			}
		}
	case PolicyPreferUser:
		fallthrough
	default:
		for _, p := range principals {
			if p != nil && p.Type == authn.PrincipalUser {
				return p, nil
			}
		}
	}
	for _, p := range principals {
		if p != nil {
			return p, nil
		}
	}
	return nil, errUnauthorized
}

func RequireAppScope(scope string) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil || p.Type != authn.PrincipalApp {
			return denyForbidden(c)
		}
		if hasAnyScope(p.Scopes, scope) {
			return c.Next()
		}
		return denyForbidden(c)
	}
}

func RequireAnyScope(scopes ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil {
			return denyForbidden(c)
		}
		if isSuperAdmin(p) {
			return c.Next()
		}
		if hasAnyScope(p.Scopes, scopes...) {
			return c.Next()
		}
		return denyForbidden(c)
	}
}

func RequireAllScopes(scopes ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := authn.PrincipalFromContext(c.Context())
		if !ok || p == nil {
			return denyForbidden(c)
		}
		if isSuperAdmin(p) {
			return c.Next()
		}
		if hasAllScopes(p.Scopes, scopes...) {
			return c.Next()
		}
		return denyForbidden(c)
	}
}

func hasAnyScope(current []string, required ...string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(current))
	for _, s := range current {
		if s != "" {
			set[s] = struct{}{}
		}
	}
	for _, need := range required {
		if need == "" {
			continue
		}
		if _, ok := set[need]; ok {
			return true
		}
	}
	return false
}

func hasAllScopes(current []string, required ...string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(current))
	for _, s := range current {
		if s != "" {
			set[s] = struct{}{}
		}
	}
	for _, need := range required {
		if need == "" {
			continue
		}
		if _, ok := set[need]; !ok {
			return false
		}
	}
	return true
}

func denyUnauthorized(c fiber.Ctx) error {
	return httpx.Unauthorized(c, "unauthorized")
}

func denyForbidden(c fiber.Ctx) error {
	return httpx.Forbidden(c, "forbidden")
}

func SetSuperAdminRoles(roles ...string) {
	normalized := normalizeStringList(roles)
	next := make(map[string]struct{}, len(normalized))
	for _, role := range normalized {
		next[role] = struct{}{}
	}

	superAdminRolesMu.Lock()
	superAdminRolesSet = next
	superAdminRolesMu.Unlock()
}

func SuperAdminRoles() []string {
	superAdminRolesMu.RLock()
	out := make([]string, 0, len(superAdminRolesSet))
	for role := range superAdminRolesSet {
		out = append(out, role)
	}
	superAdminRolesMu.RUnlock()
	sort.Strings(out)
	return out
}

func isSuperAdmin(p *authn.Principal) bool {
	if p == nil || len(p.Roles) == 0 {
		return false
	}
	superAdminRolesMu.RLock()
	defer superAdminRolesMu.RUnlock()
	for _, role := range p.Roles {
		if _, ok := superAdminRolesSet[strings.TrimSpace(role)]; ok {
			return true
		}
	}
	return false
}
