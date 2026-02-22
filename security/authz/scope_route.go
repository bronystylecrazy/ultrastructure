package authz

import "github.com/bronystylecrazy/ultrastructure/web"

// Scope adds a single scope requirement for both user and app auth schemes.
func Scope(name string) web.RouteOption {
	return Scopes(name)
}

// Scopes adds scope requirements for both user and app auth schemes.
func Scopes(names ...string) web.RouteOption {
	return func(b *web.RouteBuilder) *web.RouteBuilder {
		scopes := normalizeStringList(names)
		if len(scopes) == 0 {
			return b
		}
		b = b.Scopes(defaultUserPolicyScheme, scopes...)
		b = b.Scopes(defaultAppPolicyScheme, scopes...)
		return b
	}
}
