package authz_test

import (
	"errors"
	"testing"

	authz "github.com/bronystylecrazy/ultrastructure/security/authz"
	"github.com/bronystylecrazy/ultrastructure/web"
)

func TestValidateRouteScopes_OK(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	registry.RegisterRoute("GET", "/orders", &web.RouteMetadata{
		Security: []web.SecurityRequirement{
			{Scheme: "BearerAuth", Scopes: []string{"orders:read"}},
		},
	})

	scopeRegistry, err := authz.NewScopeRegistry(
		authz.ScopeDefinition{Name: "orders:read", Description: "Read orders"},
	)
	if err != nil {
		t.Fatalf("NewScopeRegistry: %v", err)
	}

	if err := authz.ValidateRouteScopes(registry, scopeRegistry); err != nil {
		t.Fatalf("ValidateRouteScopes: %v", err)
	}
}

func TestValidateRouteScopes_Unknown(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	registry.RegisterRoute("GET", "/orders", &web.RouteMetadata{
		Security: []web.SecurityRequirement{
			{Scheme: "BearerAuth", Scopes: []string{"orders:read"}},
		},
	})

	scopeRegistry, err := authz.NewScopeRegistry(
		authz.ScopeDefinition{Name: "orders:write", Description: "Write orders"},
	)
	if err != nil {
		t.Fatalf("NewScopeRegistry: %v", err)
	}

	err = authz.ValidateRouteScopes(registry, scopeRegistry)
	if err == nil {
		t.Fatal("expected validation error for unknown route scope")
	}
	var typed *authz.UnknownRouteScopesError
	if !errors.As(err, &typed) {
		t.Fatalf("expected UnknownRouteScopesError, got %T (%v)", err, err)
	}
	if len(typed.Items) != 1 {
		t.Fatalf("unknown items count: got=%d want=%d", len(typed.Items), 1)
	}
	if typed.Items[0].Scope != "orders:read" {
		t.Fatalf("unknown scope: got=%q want=%q", typed.Items[0].Scope, "orders:read")
	}
}

func TestBuildScopeCatalogWithRegistry_IncludesDefinitions(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	registry.RegisterRoute("GET", "/orders", &web.RouteMetadata{
		Security: []web.SecurityRequirement{
			{Scheme: "BearerAuth", Scopes: []string{"orders:read"}},
		},
	})

	scopeRegistry, err := authz.NewScopeRegistry(
		authz.ScopeDefinition{Name: "orders:read", Description: "Read orders", Owner: "team-order"},
		authz.ScopeDefinition{Name: "orders:write", Description: "Write orders", Owner: "team-order"},
	)
	if err != nil {
		t.Fatalf("NewScopeRegistry: %v", err)
	}

	catalog := authz.BuildScopeCatalogWithRegistry(registry, scopeRegistry)
	if len(catalog.ScopeDefinitions) != 2 {
		t.Fatalf("scope definitions count: got=%d want=%d", len(catalog.ScopeDefinitions), 2)
	}
	if len(catalog.Scopes) != 2 {
		t.Fatalf("scopes count: got=%d want=%d", len(catalog.Scopes), 2)
	}
	if catalog.Scopes[0] != "orders:read" || catalog.Scopes[1] != "orders:write" {
		t.Fatalf("scopes mismatch: got=%v", catalog.Scopes)
	}
}
