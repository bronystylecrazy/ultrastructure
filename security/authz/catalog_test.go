package authz_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	authn "github.com/bronystylecrazy/ultrastructure/security/authn"
	authz "github.com/bronystylecrazy/ultrastructure/security/authz"
	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func TestBuildScopeCatalog(t *testing.T) {
	container := web.NewRegistryContainer()
	reg := container.Metadata

	reg.RegisterRoute("GET", "/api/v1/orders", &web.RouteMetadata{
		OperationID: "orders_list",
		Tags:        []string{"orders"},
		Security: []web.SecurityRequirement{
			{Scheme: "BearerAuth", Scopes: []string{"orders:read"}},
		},
	})
	reg.RegisterRoute("POST", "/api/v1/orders", &web.RouteMetadata{
		OperationID: "orders_create",
		Tags:        []string{"orders"},
		Security: []web.SecurityRequirement{
			{Scheme: "BearerAuth", Scopes: []string{"orders:write"}},
			{Scheme: "ApiKeyAuth", Scopes: []string{"orders:write", "orders:read"}},
		},
	})

	catalog := authz.BuildScopeCatalog(reg)
	if len(catalog.Endpoints) != 2 {
		t.Fatalf("endpoints count: got=%d want=%d", len(catalog.Endpoints), 2)
	}
	if len(catalog.Scopes) != 2 {
		t.Fatalf("scopes count: got=%d want=%d", len(catalog.Scopes), 2)
	}
	if catalog.Scopes[0] != "orders:read" || catalog.Scopes[1] != "orders:write" {
		t.Fatalf("scopes mismatch: got=%v", catalog.Scopes)
	}
	if len(catalog.ScopeDefinitions) != 0 {
		t.Fatalf("expected empty scope definitions without governance registry, got=%d", len(catalog.ScopeDefinitions))
	}
}

func TestBuildScopeCatalogWithGovernance_IncludesPolicies(t *testing.T) {
	container := web.NewRegistryContainer()
	reg := container.Metadata

	reg.RegisterRoute("GET", "/api/v1/orders", &web.RouteMetadata{
		OperationID: "orders_list",
		Policies:    []string{"orders.read"},
		Security: []web.SecurityRequirement{
			{Scheme: "BearerAuth", Scopes: []string{"orders:read"}},
		},
	})
	policyRegistry, err := authz.NewPolicyRegistry(authz.PolicyDefinition{
		Name:          "orders.read",
		Description:   "Read orders",
		PrincipalType: authn.PrincipalUser,
		AllScopes:     []string{"orders:read"},
	})
	if err != nil {
		t.Fatalf("NewPolicyRegistry: %v", err)
	}

	catalog := authz.BuildScopeCatalogWithGovernance(reg, nil, policyRegistry)
	if len(catalog.PolicyDefinitions) != 1 {
		t.Fatalf("policy definitions count: got=%d want=%d", len(catalog.PolicyDefinitions), 1)
	}
	if len(catalog.Endpoints) != 1 {
		t.Fatalf("endpoints count: got=%d want=%d", len(catalog.Endpoints), 1)
	}
	if len(catalog.Endpoints[0].Policies) != 1 || catalog.Endpoints[0].Policies[0] != "orders.read" {
		t.Fatalf("endpoint policies mismatch: got=%v", catalog.Endpoints[0].Policies)
	}
}

func TestScopeCatalogHandler(t *testing.T) {
	container := web.NewRegistryContainer()
	reg := container.Metadata
	reg.RegisterRoute("GET", "/api/v1/items", &web.RouteMetadata{
		Security: []web.SecurityRequirement{
			{Scheme: "BearerAuth", Scopes: []string{"items:read"}},
		},
	})

	app := fiber.New()
	h := authz.NewScopeCatalogHandler(reg)
	h.Handle(web.NewRouter(app))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/authz/scopes", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got=%d want=%d", res.StatusCode, fiber.StatusOK)
	}

	meta := web.GetGlobalRegistry().GetRoute("GET", "/api/v1/authz/scopes")
	if meta == nil {
		t.Fatalf("expected route metadata for scope catalog")
	}
	okResp, exists := meta.Responses[fiber.StatusOK]
	if !exists {
		t.Fatalf("expected 200 response metadata for scope catalog")
	}
	if okResp.Type != reflect.TypeOf(authz.ScopeCatalog{}) {
		t.Fatalf("expected response type ScopeCatalog, got %v", okResp.Type)
	}
}
