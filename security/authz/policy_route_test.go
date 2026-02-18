package authz

import (
	"errors"
	"testing"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func TestPolicyRouteOptionStoresPoliciesInMetadata(t *testing.T) {
	reg, err := NewPolicyRegistry(
		PolicyDefinition{
			Name:          "orders.read",
			PrincipalType: authn.PrincipalUser,
			AllScopes:     []string{"orders:read"},
		},
		PolicyDefinition{
			Name:          "orders.write",
			PrincipalType: authn.PrincipalApp,
			AllScopes:     []string{"orders:write"},
		},
	)
	if err != nil {
		t.Fatalf("NewPolicyRegistry: %v", err)
	}
	setPolicyExpansionRegistry(reg)
	t.Cleanup(func() { setPolicyExpansionRegistry(nil) })

	registry := web.NewRegistryContainer().Metadata
	app := fiber.New()
	r := web.NewRouterWithRegistry(app, registry)

	r.Get("/orders", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) }).
		Apply(
			Policy(" orders.read "),
			Policies("orders.write"),
		)

	meta := registry.GetRoute("GET", "/orders")
	if meta == nil {
		t.Fatal("expected route metadata")
	}
	if len(meta.Policies) != 2 {
		t.Fatalf("policies count: got=%d want=%d", len(meta.Policies), 2)
	}
	if meta.Policies[0] != "orders.read" || meta.Policies[1] != "orders.write" {
		t.Fatalf("policies mismatch: got=%v", meta.Policies)
	}
	if len(meta.Security) != 2 {
		t.Fatalf("security count: got=%d want=%d", len(meta.Security), 2)
	}
}

func TestValidateRoutePolicies_Unknown(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	registry.RegisterRoute("GET", "/orders", &web.RouteMetadata{
		Policies: []string{"orders.read", "orders.unknown"},
	})
	policyRegistry, err := NewPolicyRegistry(PolicyDefinition{Name: "orders.read"})
	if err != nil {
		t.Fatalf("NewPolicyRegistry: %v", err)
	}

	err = ValidateRoutePolicies(registry, policyRegistry)
	if err == nil {
		t.Fatal("expected unknown policy validation error")
	}
	var typed *UnknownRoutePoliciesError
	if !errors.As(err, &typed) {
		t.Fatalf("expected UnknownRoutePoliciesError, got %T (%v)", err, err)
	}
	if len(typed.Items) != 1 {
		t.Fatalf("unknown policy count: got=%d want=%d", len(typed.Items), 1)
	}
}

func TestExpandRoutePolicies_AddsSecurityRequirements(t *testing.T) {
	registry := web.NewRegistryContainer().Metadata
	registry.RegisterRoute("GET", "/orders", &web.RouteMetadata{
		Policies: []string{"orders.read", "orders.write.any"},
	})
	policyRegistry, err := NewPolicyRegistry(
		PolicyDefinition{
			Name:          "orders.read",
			PrincipalType: authn.PrincipalUser,
			AllScopes:     []string{"orders:read"},
		},
		PolicyDefinition{
			Name:          "orders.write.any",
			PrincipalType: authn.PrincipalApp,
			AnyScopes:     []string{"orders:write", "orders:admin"},
		},
	)
	if err != nil {
		t.Fatalf("NewPolicyRegistry: %v", err)
	}

	ExpandRoutePolicies(registry, policyRegistry)

	meta := registry.GetRoute("GET", "/orders")
	if meta == nil {
		t.Fatal("expected route metadata")
	}
	if len(meta.Security) != 3 {
		t.Fatalf("security requirements count: got=%d want=%d", len(meta.Security), 3)
	}

	got := map[string]struct{}{}
	for _, req := range meta.Security {
		key := req.Scheme + ":" + join(req.Scopes)
		got[key] = struct{}{}
	}
	want := []string{
		"BearerAuth:orders:read",
		"ApiKeyAuth:orders:admin",
		"ApiKeyAuth:orders:write",
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Fatalf("missing expanded requirement: %s got=%v", k, got)
		}
	}
}

func join(in []string) string {
	if len(in) == 0 {
		return ""
	}
	out := in[0]
	for i := 1; i < len(in); i++ {
		out += "," + in[i]
	}
	return out
}
