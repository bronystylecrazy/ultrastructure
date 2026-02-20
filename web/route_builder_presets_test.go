package web

import (
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRouteBuilder_StandardErrorsAddsCommonResponses(t *testing.T) {
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/preset", func(c fiber.Ctx) error { return c.SendStatus(200) }).StandardErrors()

	meta := registry.GetRoute("GET", "/preset")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	for _, code := range []int{400, 401, 403, 404, 500} {
		resp, ok := meta.Responses[code]
		if !ok {
			t.Fatalf("expected response %d to be present", code)
		}
		if resp.Type != reflect.TypeOf(Error{}) {
			t.Fatalf("expected response %d type Error, got %v", code, resp.Type)
		}
	}
}

func TestRouteBuilder_ValidationErrorsAddsValidationResponses(t *testing.T) {
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Post("/validate", func(c fiber.Ctx) error { return c.SendStatus(201) }).ValidationErrors()

	meta := registry.GetRoute("POST", "/validate")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}

	for _, code := range []int{400, 422} {
		resp, ok := meta.Responses[code]
		if !ok {
			t.Fatalf("expected response %d to be present", code)
		}
		if resp.Type != reflect.TypeOf(Error{}) {
			t.Fatalf("expected response %d type Error, got %v", code, resp.Type)
		}
	}
}

func TestRouteBuilder_StandardErrorsDoesNotOverrideExplicitResponse(t *testing.T) {
	registry := NewMetadataRegistry()

	type customNotFound struct {
		Reason string `json:"reason"`
	}

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/custom", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Produces(customNotFound{}, 404).
		StandardErrors()

	meta := registry.GetRoute("GET", "/custom")
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
	registry := NewMetadataRegistry()

	type user struct {
		ID string `json:"id"`
	}

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/users", func(c fiber.Ctx) error { return c.SendStatus(200) }).Paginated(user{})

	meta := registry.GetRoute("GET", "/users")
	if meta == nil || meta.Pagination == nil {
		t.Fatalf("expected pagination metadata")
	}
	if meta.Pagination.ItemType != reflect.TypeOf(user{}) {
		t.Fatalf("unexpected pagination item type: %v", meta.Pagination.ItemType)
	}
}

func TestRouteBuilder_BodyAtLeastOneSetsRequestBodyConstraint(t *testing.T) {
	registry := NewMetadataRegistry()

	type patchUser struct {
		Name *string `json:"name,omitempty"`
	}

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Patch("/users/:id", func(c fiber.Ctx) error { return c.SendStatus(200) }).BodyAtLeastOne(patchUser{})

	meta := registry.GetRoute("PATCH", "/users/:id")
	if meta == nil || meta.RequestBody == nil {
		t.Fatalf("expected request body metadata")
	}
	if meta.RequestBody.RequireAtLeastOne != true {
		t.Fatalf("expected RequireAtLeastOne=true")
	}
}

func TestRouteBuilder_WithComposesRouteOptions(t *testing.T) {
	registry := NewMetadataRegistry()

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
	router := NewRouterWithRegistry(app, registry)
	router.Get("/apply", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		With(withUsersTag(), withAuthHeader())

	meta := registry.GetRoute("GET", "/apply")
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

func TestRouteBuilder_WithComposesMetadataRouteOptions(t *testing.T) {
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/apply-meta", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		With(
			Name("GetApplyMeta"),
			Tag("System"),
			Tags("Users"),
			func(b *RouteBuilder) *RouteBuilder { return b.Summary("Get apply metadata") },
			func(b *RouteBuilder) *RouteBuilder {
				return b.Description("Demonstrates Name/Tag/Summary/Description route options")
			},
		)

	meta := registry.GetRoute("GET", "/apply-meta")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	if meta.OperationID != "GetApplyMeta" {
		t.Fatalf("expected operationId GetApplyMeta, got %q", meta.OperationID)
	}
	if meta.Summary != "Get apply metadata" {
		t.Fatalf("expected summary to be set, got %q", meta.Summary)
	}
	if meta.Description != "Demonstrates Name/Tag/Summary/Description route options" {
		t.Fatalf("expected description to be set, got %q", meta.Description)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "System" || meta.Tags[1] != "Users" {
		t.Fatalf("expected tags [System Users], got %v", meta.Tags)
	}
}

func TestRouteBuilder_TaggedName(t *testing.T) {
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/users/:id", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Tags("Users").
		TaggedName("GetUserByID")

	meta := registry.GetRoute("GET", "/users/:id")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	if meta.OperationID != "Users_GetUserByID" {
		t.Fatalf("expected operationId Users_GetUserByID, got %q", meta.OperationID)
	}
}

func TestRouteBuilder_WithTaggedName(t *testing.T) {
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/users/:id", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		With(
			Tag("Users"),
			TaggedName("GetUserByID"),
		)

	meta := registry.GetRoute("GET", "/users/:id")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	if meta.OperationID != "Users_GetUserByID" {
		t.Fatalf("expected operationId Users_GetUserByID, got %q", meta.OperationID)
	}
}

func TestRouteBuilder_HeaderAndCookieExtStoreParameterExtensions(t *testing.T) {
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/ext-params", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		HeaderRequiredExt("X-Tenant-ID", "", "x-nullable,x-owner=platform,!x-omitempty", "Tenant header").
		CookieExt("session_id", "", "x-format=legacy", "Session cookie")

	meta := registry.GetRoute("GET", "/ext-params")
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

func TestRouteBuilder_WithGenericRouteOptions(t *testing.T) {
	registry := NewMetadataRegistry()

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
	router := NewRouterWithRegistry(app, registry)
	router.Post("/apply-generic", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		With(
			func(b *RouteBuilder) *RouteBuilder { return b.Body(createReq{}) },
			func(b *RouteBuilder) *RouteBuilder { return b.Query(listQ{}) },
			func(b *RouteBuilder) *RouteBuilder { return b.Produces(userResp{}, 200) },
			func(b *RouteBuilder) *RouteBuilder { return b.Produces(userResp{}, 201) },
		)

	meta := registry.GetRoute("POST", "/apply-generic")
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
	registry := NewMetadataRegistry()

	type okResp struct{ OK bool }
	type createdResp struct{ ID string }
	type errResp struct{ Error string }

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Post("/status-methods", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		Ok(okResp{}, "Success payload").
		Create(createdResp{}, "Created payload").
		Conflict(errResp{}, "Resource conflict").
		NotFound(errResp{}).
		InternalError(errResp{})

	meta := registry.GetRoute("POST", "/status-methods")
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

func TestRouteBuilder_WithStatusGenericOptions(t *testing.T) {
	registry := NewMetadataRegistry()

	type createdResp struct{ ID string }
	type errResp struct{ Error string }

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Post("/status-options", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		With(
			func(b *RouteBuilder) *RouteBuilder {
				return b.ProducesWithDescription(createdResp{}, 201, "Created from options")
			},
			func(b *RouteBuilder) *RouteBuilder {
				return b.ProducesWithDescription(errResp{}, 409, "Conflict from options")
			},
			func(b *RouteBuilder) *RouteBuilder { return b.Produces(errResp{}, 404) },
			func(b *RouteBuilder) *RouteBuilder { return b.Produces(errResp{}, 500) },
		)

	meta := registry.GetRoute("POST", "/status-options")
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
	registry := NewMetadataRegistry()

	type okResp struct {
		OK bool `json:"ok"`
	}

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/response-headers", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Ok(okResp{}).
		SetHeaders(200, "X-Request-ID", "", "Request correlation ID").
		SetCookies(200, "session_id")

	meta := registry.GetRoute("GET", "/response-headers")
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

func TestRouteBuilder_OkChainTracksMultipleContentTypesOnSameStatus(t *testing.T) {
	registry := NewMetadataRegistry()

	type jsonResp struct {
		OK bool `json:"ok"`
	}

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/multi-content", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Ok("").
		Ok(jsonResp{})

	meta := registry.GetRoute("GET", "/multi-content")
	if meta == nil {
		t.Fatalf("expected route metadata")
	}
	resp := meta.Responses[200]
	if resp.Content == nil {
		t.Fatalf("expected 200 response content map")
	}
	if got := resp.Content["text/plain"]; got != reflect.TypeOf("") {
		t.Fatalf("expected text/plain type string, got %v", got)
	}
	if got := resp.Content["application/json"]; got != reflect.TypeOf(jsonResp{}) {
		t.Fatalf("expected application/json type jsonResp, got %v", got)
	}
}

func TestRouteBuilder_RequestBodyChainTracksMultipleContentTypes(t *testing.T) {
	registry := NewMetadataRegistry()

	type jsonReq struct {
		Name string `json:"name"`
	}
	type formReq struct {
		Name string `form:"name"`
	}

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Post("/multi-body", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Body(jsonReq{}).
		Form(formReq{})

	meta := registry.GetRoute("POST", "/multi-body")
	if meta == nil || meta.RequestBody == nil {
		t.Fatalf("expected request body metadata")
	}
	if meta.RequestBody.Content == nil {
		t.Fatalf("expected request body content map")
	}
	if got := meta.RequestBody.Content["application/json"]; got != reflect.TypeOf(jsonReq{}) {
		t.Fatalf("expected application/json body type jsonReq, got %v", got)
	}
	if got := meta.RequestBody.Content["application/x-www-form-urlencoded"]; got != reflect.TypeOf(formReq{}) {
		t.Fatalf("expected form body type formReq, got %v", got)
	}
}

func TestRouteBuilder_PluralHeadersAndCookiesUseMap(t *testing.T) {
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	router.Get("/map-params", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Headers(map[string]any{
			"X-Request-ID": "",
			"X-Tenant-ID":  "",
		}).
		CookiesRequired(map[string]any{
			"session_id": "",
		})

	meta := registry.GetRoute("GET", "/map-params")
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
	registry := NewMetadataRegistry()

	app := fiber.New()
	router := NewRouterWithRegistry(app, registry)
	users := router.Group("/users")

	users.Post("", func(c fiber.Ctx) error { return c.SendStatus(201) }).
		Name("CreateUser")
	users.Get("/:id", func(c fiber.Ctx) error { return c.SendStatus(200) }).
		Name("GetUserByID")

	if got := registry.GetRoute("POST", "/users"); got == nil {
		t.Fatalf("expected metadata key POST:/users to exist")
	}
	if got := registry.GetRoute("GET", "/users/:id"); got == nil {
		t.Fatalf("expected metadata key GET:/users/:id to exist")
	}
	if got := registry.GetRoute("POST", ""); got != nil {
		t.Fatalf("did not expect relative metadata key POST:\"\"")
	}
}
