package autoswag

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type goldenRole string

const (
	goldenRoleAdmin goldenRole = "admin"
	goldenRoleUser  goldenRole = "user"
)

type goldenListUsersQuery struct {
	Limit  int        `query:"limit" default:"20" example:"50"`
	Status goldenRole `query:"status"`
}

type goldenUser struct {
	ID    string     `json:"id" example:"usr_123"`
	Email string     `json:"email" example:"user@example.com"`
	Role  goldenRole `json:"role"`
}

type goldenCreateUserBody struct {
	Email string     `json:"email" validate:"required,email"`
	Role  goldenRole `json:"role" default:"user"`
}

type goldenPatchUserBody struct {
	Email *string `json:"email,omitempty"`
	Role  *string `json:"role,omitempty"`
}

func TestBuildOpenAPISpec_Golden(t *testing.T) {
	GetGlobalRegistry().Clear()
	ClearEnumRegistry()
	ClearSchemaNameRegistry()
	ClearOperationIDHook()
	ClearOperationIDTagPrefix()
	ClearGlobalHook()
	defer GetGlobalRegistry().Clear()
	defer ClearEnumRegistry()
	defer ClearSchemaNameRegistry()
	defer ClearOperationIDHook()
	defer ClearOperationIDTagPrefix()
	defer ClearGlobalHook()

	RegisterEnum(goldenRoleAdmin, goldenRoleUser)

	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{
		Tags:      []string{"Users"},
		QueryType: reflect.TypeOf(goldenListUsersQuery{}),
		Pagination: &PaginationMetadata{
			ItemType: reflect.TypeOf(goldenUser{}),
		},
		Security: []SecurityRequirement{
			{Scheme: "BearerAuth"},
		},
	})

	GetGlobalRegistry().RegisterRoute("POST", "/users", &RouteMetadata{
		Tags: []string{"Users"},
		RequestBody: &RequestBodyMetadata{
			Type:         reflect.TypeOf(goldenCreateUserBody{}),
			Required:     true,
			ContentTypes: []string{"application/json"},
		},
		Responses: map[int]ResponseMetadata{
			201: {Type: reflect.TypeOf(goldenUser{}), ContentType: "application/json"},
		},
	})

	GetGlobalRegistry().RegisterRoute("PATCH", "/users/:id", &RouteMetadata{
		Tags: []string{"Users"},
		RequestBody: &RequestBodyMetadata{
			Type:              reflect.TypeOf(goldenPatchUserBody{}),
			Required:          true,
			ContentTypes:      []string{"application/json"},
			RequireAtLeastOne: true,
		},
		Parameters: []ParameterMetadata{
			{Name: "X-Tenant-ID", In: "header", Type: reflect.TypeOf(""), Required: true},
		},
		Security: []SecurityRequirement{},
		Responses: map[int]ResponseMetadata{
			200: {Type: reflect.TypeOf(goldenUser{}), ContentType: "application/json"},
		},
	})

	GetGlobalRegistry().RegisterRoute("GET", "/health", &RouteMetadata{
		Tags:    []string{"System"},
		Summary: "Health check",
		Responses: map[int]ResponseMetadata{
			200: {Type: reflect.TypeOf(map[string]string{}), ContentType: "application/json"},
		},
	})

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/users"},
		{Method: "POST", Path: "/users"},
		{Method: "PATCH", Path: "/users/:id"},
		{Method: "GET", Path: "/health"},
	}, Config{Name: "Golden API"}, OpenAPIBuildOptions{
		SecuritySchemes: map[string]interface{}{
			"BearerAuth": map[string]interface{}{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
			},
		},
		DefaultSecurity: []SecurityRequirement{
			{Scheme: "BearerAuth"},
		},
		TagDescriptions: map[string]string{
			"Users":  "User management",
			"System": "System endpoints",
		},
		TermsOfService: "https://example.com/terms",
		Contact: &OpenAPIContact{
			Name:  "API Team",
			Email: "api@example.com",
		},
		License: &OpenAPILicense{
			Name: "Private Use Only",
		},
	})

	actual, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	actual = append(actual, '\n')

	goldenPath := filepath.Join("testdata", "openapi.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, actual, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	if string(expected) != string(actual) {
		t.Fatalf("golden mismatch for %s (run with UPDATE_GOLDEN=1 to update)", goldenPath)
	}
}
