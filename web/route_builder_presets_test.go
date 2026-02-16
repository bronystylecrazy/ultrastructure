package web

import (
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRouteBuilder_StandardErrorsAddsCommonResponses(t *testing.T) {
	GetGlobalRegistry().Clear()

	app := fiber.New()
	router := NewRouter(app)
	router.Get("/preset", func(c fiber.Ctx) error { return c.SendStatus(200) }).StandardErrors()

	meta := GetGlobalRegistry().GetRoute("GET", "/preset")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	for _, code := range []int{400, 401, 403, 404, 500} {
		resp, ok := meta.Responses[code]
		if !ok {
			t.Fatalf("expected response %d to be present", code)
		}
		if resp.Type != reflect.TypeOf(OpenAPIErrorResponse{}) {
			t.Fatalf("expected response %d type OpenAPIErrorResponse, got %v", code, resp.Type)
		}
	}
}

func TestRouteBuilder_ValidationErrorsAddsValidationResponses(t *testing.T) {
	GetGlobalRegistry().Clear()

	app := fiber.New()
	router := NewRouter(app)
	router.Post("/validate", func(c fiber.Ctx) error { return c.SendStatus(201) }).ValidationErrors()

	meta := GetGlobalRegistry().GetRoute("POST", "/validate")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	for _, code := range []int{400, 422} {
		resp, ok := meta.Responses[code]
		if !ok {
			t.Fatalf("expected response %d to be present", code)
		}
		if resp.Type != reflect.TypeOf(OpenAPIErrorResponse{}) {
			t.Fatalf("expected response %d type OpenAPIErrorResponse, got %v", code, resp.Type)
		}
	}
}

func TestRouteBuilder_StandardErrorsDoesNotOverrideExplicitResponse(t *testing.T) {
	GetGlobalRegistry().Clear()

	type customNotFound struct {
		Reason string `json:"reason"`
	}

	app := fiber.New()
	router := NewRouter(app)
	router.Get("/custom", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Produces(customNotFound{}, 404).
		StandardErrors()

	meta := GetGlobalRegistry().GetRoute("GET", "/custom")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	resp, ok := meta.Responses[404]
	if !ok {
		t.Fatalf("expected response 404 to be present")
	}
	if resp.Type != reflect.TypeOf(customNotFound{}) {
		t.Fatalf("expected custom 404 type to be preserved, got %v", resp.Type)
	}
}

func TestRouteBuilder_PaginatedSetsPaginationMetadata(t *testing.T) {
	GetGlobalRegistry().Clear()

	type user struct {
		ID string `json:"id"`
	}

	app := fiber.New()
	router := NewRouter(app)
	router.Get("/users", func(c fiber.Ctx) error { return c.SendStatus(200) }).Paginated(user{})

	meta := GetGlobalRegistry().GetRoute("GET", "/users")
	if meta == nil || meta.Pagination == nil {
		t.Fatalf("expected pagination metadata")
	}
	if meta.Pagination.ItemType != reflect.TypeOf(user{}) {
		t.Fatalf("unexpected pagination item type: %v", meta.Pagination.ItemType)
	}
}

func TestRouteBuilder_BodyAtLeastOneSetsRequestBodyConstraint(t *testing.T) {
	GetGlobalRegistry().Clear()

	type patchUser struct {
		Name *string `json:"name,omitempty"`
	}

	app := fiber.New()
	router := NewRouter(app)
	router.Patch("/users/:id", func(c fiber.Ctx) error { return c.SendStatus(200) }).BodyAtLeastOne(patchUser{})

	meta := GetGlobalRegistry().GetRoute("PATCH", "/users/:id")
	if meta == nil || meta.RequestBody == nil {
		t.Fatalf("expected request body metadata")
	}
	if meta.RequestBody.RequireAtLeastOne != true {
		t.Fatalf("expected RequireAtLeastOne=true")
	}
}

func TestRouteBuilder_ApplyComposesRouteOptions(t *testing.T) {
	GetGlobalRegistry().Clear()

	withUsersTag := func() RouteOption {
		return func(b *RouteBuilder) *RouteBuilder {
			return b.Tags("Users")
		}
	}
	withAuthHeader := func() RouteOption {
		return func(b *RouteBuilder) *RouteBuilder {
			return b.HeaderRequired("X-Tenant-ID", "", "Tenant header")
		}
	}

	app := fiber.New()
	router := NewRouter(app)
	router.Get("/apply", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Apply(withUsersTag(), withAuthHeader())

	meta := GetGlobalRegistry().GetRoute("GET", "/apply")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	if len(meta.Tags) == 0 || meta.Tags[0] != "Users" {
		t.Fatalf("expected Users tag, got %v", meta.Tags)
	}

	foundHeader := false
	for _, p := range meta.Parameters {
		if p.In == "header" && p.Name == "X-Tenant-ID" && p.Required && p.Description == "Tenant header" {
			foundHeader = true
			break
		}
	}
	if !foundHeader {
		t.Fatalf("expected required X-Tenant-ID header parameter")
	}
}

func TestRouteBuilder_HeaderAndCookieExtStoreParameterExtensions(t *testing.T) {
	GetGlobalRegistry().Clear()

	app := fiber.New()
	router := NewRouter(app)
	router.Get("/ext-params", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		HeaderRequiredExt("X-Tenant-ID", "", "x-nullable,x-owner=platform,!x-omitempty", "Tenant header").
		CookieExt("session_id", "", "x-format=legacy", "Session cookie")

	meta := GetGlobalRegistry().GetRoute("GET", "/ext-params")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	var headerOK, cookieOK bool
	for _, p := range meta.Parameters {
		if p.In == "header" && p.Name == "X-Tenant-ID" && p.Extensions == "x-nullable,x-owner=platform,!x-omitempty" {
			headerOK = true
		}
		if p.In == "cookie" && p.Name == "session_id" && p.Extensions == "x-format=legacy" {
			cookieOK = true
		}
	}
	if !headerOK {
		t.Fatalf("expected header extensions to be stored in metadata")
	}
	if !cookieOK {
		t.Fatalf("expected cookie extensions to be stored in metadata")
	}
}

func TestRouteBuilder_ApplyWithGenericRouteOptions(t *testing.T) {
	GetGlobalRegistry().Clear()

	type createReq struct {
		Name string `json:"name"`
	}
	type userResp struct {
		ID string `json:"id"`
	}
	type listQ struct {
		Limit int `query:"limit"`
	}

	app := fiber.New()
	router := NewRouter(app)
	router.Post("/apply-generic", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		Apply(
			Body(createReq{}),
			Query[listQ](),
			Produce[userResp](),    // default 200
			Produce[userResp](201), // explicit status
		)

	meta := GetGlobalRegistry().GetRoute("POST", "/apply-generic")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	if meta.RequestBody == nil || meta.RequestBody.Type != reflect.TypeOf(createReq{}) {
		t.Fatalf("expected request body type createReq, got %+v", meta.RequestBody)
	}
	if meta.QueryType != reflect.TypeOf(listQ{}) {
		t.Fatalf("expected query type listQ, got %v", meta.QueryType)
	}
	if meta.Responses[200].Type != reflect.TypeOf(userResp{}) {
		t.Fatalf("expected 200 response type userResp, got %v", meta.Responses[200].Type)
	}
	if meta.Responses[201].Type != reflect.TypeOf(userResp{}) {
		t.Fatalf("expected 201 response type userResp, got %v", meta.Responses[201].Type)
	}
}

func TestRouteBuilder_StatusConvenienceMethods(t *testing.T) {
	GetGlobalRegistry().Clear()

	type okResp struct{ OK bool }
	type createdResp struct{ ID string }
	type errResp struct{ Error string }

	app := fiber.New()
	router := NewRouter(app)
	router.Post("/status-methods", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		Ok(okResp{}, "Success payload").
		Create(createdResp{}, "Created payload").
		Conflict(errResp{}, "Resource conflict").
		NotFound(errResp{}).
		InternalError(errResp{})

	meta := GetGlobalRegistry().GetRoute("POST", "/status-methods")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	if meta.Responses[200].Type != reflect.TypeOf(okResp{}) {
		t.Fatalf("expected 200 type okResp, got %v", meta.Responses[200].Type)
	}
	if meta.Responses[200].Description != "Success payload" {
		t.Fatalf("expected custom 200 description, got %q", meta.Responses[200].Description)
	}
	if meta.Responses[201].Type != reflect.TypeOf(createdResp{}) {
		t.Fatalf("expected 201 type createdResp, got %v", meta.Responses[201].Type)
	}
	if meta.Responses[201].Description != "Created payload" {
		t.Fatalf("expected custom 201 description, got %q", meta.Responses[201].Description)
	}
	if meta.Responses[404].Type != reflect.TypeOf(errResp{}) {
		t.Fatalf("expected 404 type errResp, got %v", meta.Responses[404].Type)
	}
	if meta.Responses[409].Type != reflect.TypeOf(errResp{}) {
		t.Fatalf("expected 409 type errResp, got %v", meta.Responses[409].Type)
	}
	if meta.Responses[409].Description != "Resource conflict" {
		t.Fatalf("expected custom 409 description, got %q", meta.Responses[409].Description)
	}
	if meta.Responses[500].Type != reflect.TypeOf(errResp{}) {
		t.Fatalf("expected 500 type errResp, got %v", meta.Responses[500].Type)
	}
}

func TestRouteBuilder_ApplyWithStatusGenericOptions(t *testing.T) {
	GetGlobalRegistry().Clear()

	type createdResp struct{ ID string }
	type errResp struct{ Error string }

	app := fiber.New()
	router := NewRouter(app)
	router.Post("/status-options", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		Apply(
			Create[createdResp]("Created from options"),
			Conflict[errResp]("Conflict from options"),
			NotFound[errResp](),
			InternalError[errResp](),
		)

	meta := GetGlobalRegistry().GetRoute("POST", "/status-options")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	if meta.Responses[201].Type != reflect.TypeOf(createdResp{}) {
		t.Fatalf("expected 201 type createdResp, got %v", meta.Responses[201].Type)
	}
	if meta.Responses[201].Description != "Created from options" {
		t.Fatalf("expected custom 201 description from options, got %q", meta.Responses[201].Description)
	}
	if meta.Responses[404].Type != reflect.TypeOf(errResp{}) {
		t.Fatalf("expected 404 type errResp, got %v", meta.Responses[404].Type)
	}
	if meta.Responses[409].Type != reflect.TypeOf(errResp{}) {
		t.Fatalf("expected 409 type errResp, got %v", meta.Responses[409].Type)
	}
	if meta.Responses[409].Description != "Conflict from options" {
		t.Fatalf("expected custom 409 description from options, got %q", meta.Responses[409].Description)
	}
	if meta.Responses[500].Type != reflect.TypeOf(errResp{}) {
		t.Fatalf("expected 500 type errResp, got %v", meta.Responses[500].Type)
	}
}

func TestRouteBuilder_SetHeadersAndSetCookiesMetadata(t *testing.T) {
	GetGlobalRegistry().Clear()

	type okResp struct {
		OK bool `json:"ok"`
	}

	app := fiber.New()
	router := NewRouter(app)
	router.Get("/response-headers", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Ok(okResp{}).
		SetHeaders(200, "X-Request-ID", "", "Request correlation ID").
		SetCookies(200, "session_id")

	meta := GetGlobalRegistry().GetRoute("GET", "/response-headers")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	resp := meta.Responses[200]
	if resp.Headers == nil {
		t.Fatalf("expected response headers metadata")
	}
	if _, ok := resp.Headers["X-Request-ID"]; !ok {
		t.Fatalf("expected X-Request-ID response header")
	}
	if _, ok := resp.Headers["Set-Cookie"]; !ok {
		t.Fatalf("expected Set-Cookie response header")
	}
}

func TestRouteBuilder_PluralHeadersAndCookiesUseMap(t *testing.T) {
	GetGlobalRegistry().Clear()

	app := fiber.New()
	router := NewRouter(app)
	router.Get("/map-params", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Headers(map[string]any{
			"X-Request-ID": "",
			"X-Tenant-ID":  "",
		}).
		CookiesRequired(map[string]any{
			"session_id": "",
		})

	meta := GetGlobalRegistry().GetRoute("GET", "/map-params")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	var hasReqID, hasTenant, hasSession bool
	for _, p := range meta.Parameters {
		switch p.In + ":" + p.Name {
		case "header:X-Request-ID":
			hasReqID = !p.Required
		case "header:X-Tenant-ID":
			hasTenant = !p.Required
		case "cookie:session_id":
			hasSession = p.Required
		}
	}
	if !hasReqID || !hasTenant || !hasSession {
		t.Fatalf("expected headers/cookies from map methods, got %+v", meta.Parameters)
	}
}

func TestRouteBuilder_GroupRoutesRegisterFullPathMetadata(t *testing.T) {
	GetGlobalRegistry().Clear()

	app := fiber.New()
	router := NewRouter(app)
	users := router.Group("/users")

	users.Post("", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		Name("CreateUser")
	users.Get("/:id", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Name("GetUserByID")

	if got := GetGlobalRegistry().GetRoute("POST", "/users"); got == nil {
		t.Fatalf("expected metadata key POST:/users to exist")
	}
	if got := GetGlobalRegistry().GetRoute("GET", "/users/:id"); got == nil {
		t.Fatalf("expected metadata key GET:/users/:id to exist")
	}
	if got := GetGlobalRegistry().GetRoute("POST", ""); got != nil {
		t.Fatalf("did not expect relative metadata key POST:\"\"")
	}
}
