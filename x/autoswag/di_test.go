package autoswag

import "testing"

func TestFilterRoutesByPrefix(t *testing.T) {
	routes := []RouteInfo{
		{Method: "GET", Path: "/api/v1/users"},
		{Method: "GET", Path: "/api/v1/users/:id"},
		{Method: "GET", Path: "/api/v2/users"},
		{Method: "GET", Path: "/health"},
	}

	filtered := filterRoutesByPrefix(routes, "/api/v1/")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 routes for /api/v1 prefix, got %d", len(filtered))
	}
	for _, route := range filtered {
		if route.Path != "/api/v1/users" && route.Path != "/api/v1/users/:id" {
			t.Fatalf("unexpected filtered route: %s", route.Path)
		}
	}
}

func TestWithVersionedDocs_NormalizesValues(t *testing.T) {
	cfg := ResolveOptions("/docs", WithVersionedDocs("docs/v1/", "api/v1/", "API v1"))
	if len(cfg.VersionedDocs) != 1 {
		t.Fatalf("expected one versioned docs entry, got %d", len(cfg.VersionedDocs))
	}
	doc := cfg.VersionedDocs[0]
	if doc.Path != "/docs/v1" {
		t.Fatalf("expected normalized docs path /docs/v1, got %q", doc.Path)
	}
	if doc.Prefix != "/api/v1" {
		t.Fatalf("expected normalized route prefix /api/v1, got %q", doc.Prefix)
	}
	if doc.Name != "API v1" {
		t.Fatalf("expected name API v1, got %q", doc.Name)
	}
}
