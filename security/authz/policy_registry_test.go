package authz_test

import (
	"testing"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	authz "github.com/bronystylecrazy/ultrastructure/security/authz"
)

func TestNewPolicyRegistry_NormalizesDefinition(t *testing.T) {
	reg, err := authz.NewPolicyRegistry(authz.PolicyDefinition{
		Name:          "  orders.read  ",
		Description:   "  Read orders  ",
		Owner:         "  team-orders ",
		PrincipalType: authn.PrincipalUser,
		AnyScopes:     []string{"orders:read", "orders:read", " ", "orders:view"},
		AllScopes:     []string{"account:active", "", "account:active"},
		Roles:         []string{"admin", "admin", " "},
	})
	if err != nil {
		t.Fatalf("NewPolicyRegistry: %v", err)
	}
	def, ok := reg.Get("orders.read")
	if !ok {
		t.Fatal("expected policy orders.read")
	}
	if def.Description != "Read orders" {
		t.Fatalf("description mismatch: got=%q", def.Description)
	}
	if def.Owner != "team-orders" {
		t.Fatalf("owner mismatch: got=%q", def.Owner)
	}
	if len(def.AnyScopes) != 2 || def.AnyScopes[0] != "orders:read" || def.AnyScopes[1] != "orders:view" {
		t.Fatalf("any scopes mismatch: got=%v", def.AnyScopes)
	}
	if len(def.AllScopes) != 1 || def.AllScopes[0] != "account:active" {
		t.Fatalf("all scopes mismatch: got=%v", def.AllScopes)
	}
	if len(def.Roles) != 1 || def.Roles[0] != "admin" {
		t.Fatalf("roles mismatch: got=%v", def.Roles)
	}
}

func TestNewPolicyRegistry_ConflictingDuplicateReturnsError(t *testing.T) {
	_, err := authz.NewPolicyRegistry(
		authz.PolicyDefinition{
			Name:      "orders.read",
			AnyScopes: []string{"orders:read"},
		},
		authz.PolicyDefinition{
			Name:      "orders.read",
			AnyScopes: []string{"orders:view"},
		},
	)
	if err == nil {
		t.Fatal("expected error for conflicting duplicate policy")
	}
}

func TestPolicyRegistry_HasAndAll(t *testing.T) {
	reg, err := authz.NewPolicyRegistry(
		authz.PolicyDefinition{Name: "orders.read"},
		authz.PolicyDefinition{Name: "orders.write"},
	)
	if err != nil {
		t.Fatalf("NewPolicyRegistry: %v", err)
	}
	if !reg.Has("orders.read") {
		t.Fatal("expected Has(orders.read)=true")
	}
	if reg.Has("unknown.policy") {
		t.Fatal("expected Has(unknown.policy)=false")
	}
	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("all count: got=%d want=%d", len(all), 2)
	}
	if all[0].Name != "orders.read" || all[1].Name != "orders.write" {
		t.Fatalf("all order mismatch: got=%v,%v", all[0].Name, all[1].Name)
	}
}
