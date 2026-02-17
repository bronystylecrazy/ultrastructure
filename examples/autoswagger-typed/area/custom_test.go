package custom

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
)

func newTestApp() *fiber.App {
	app := fiber.New()
	router := web.NewRouter(app)
	NewUserHandler().Handle(router)
	return app
}

func TestCreateUserPayloadValidation(t *testing.T) {
	app := newTestApp()

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing_name",
			body:       map[string]interface{}{"email": "john@example.com"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing_email",
			body:       map[string]interface{}{"name": "John"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid_payload",
			body:       map[string]interface{}{"name": "John", "email": "john@example.com", "age": 20},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "age_must_be_less_than_150",
			body:       map[string]interface{}{"name": "John", "email": "john@example.com", "age": 150},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			res, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}

			if res.StatusCode != tc.wantStatus {
				t.Fatalf("status: got %d want %d", res.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestUpdateUserPayloadValidation(t *testing.T) {
	app := newTestApp()

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "empty_payload",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid_partial_payload",
			body:       map[string]interface{}{"name": "Updated Name", "age": 42},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid_age_payload",
			body:       map[string]interface{}{"age": 151},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}

			req := httptest.NewRequest(http.MethodPut, "/users/123", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			res, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}

			if res.StatusCode != tc.wantStatus {
				t.Fatalf("status: got %d want %d", res.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestOpenAPIIncludesValidateTagConstraintsForPayload(t *testing.T) {
	web.GetGlobalRegistry().Clear()
	defer web.GetGlobalRegistry().Clear()

	app := newTestApp()
	routes := web.InspectFiberRoutes(app, nil)
	spec := web.BuildOpenAPISpec(routes, web.Config{Name: "Typed Example API"})

	postOp, ok := spec.Paths["/users"]["post"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected POST /users operation")
	}

	requestBody, ok := postOp["requestBody"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected requestBody in POST /users")
	}
	content := requestBody["content"].(map[string]interface{})
	jsonMedia := content["application/json"].(map[string]interface{})
	schema := jsonMedia["schema"].(map[string]interface{})
	if schema["$ref"] != "#/components/schemas/CreateUserRequest" {
		t.Fatalf("expected request schema ref CreateUserRequest, got %v", schema["$ref"])
	}

	createSchema, ok := spec.Components.Schemas["CreateUserRequest"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected CreateUserRequest schema in components")
	}
	props := createSchema["properties"].(map[string]interface{})

	nameSchema := props["name"].(map[string]interface{})
	if nameSchema["type"] != "string" {
		t.Fatalf("expected name to be string, got %v", nameSchema["type"])
	}

	emailSchema := props["email"].(map[string]interface{})
	if emailSchema["format"] != "email" {
		t.Fatalf("expected email format=email, got %v", emailSchema["format"])
	}

	ageSchema := props["age"].(map[string]interface{})
	if ageSchema["maximum"] != float64(150) || ageSchema["exclusiveMaximum"] != true {
		t.Fatalf("expected age lt=150 constraints, got %v", ageSchema)
	}

	required := createSchema["required"].([]string)
	requiredSet := map[string]bool{}
	for _, field := range required {
		requiredSet[field] = true
	}
	if !requiredSet["name"] || !requiredSet["email"] {
		t.Fatalf("expected required fields name and email, got %v", required)
	}
}
