package web

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"
)

type listUsersQuery struct {
	Page      int       `query:"page"`
	Search    string    `query:"search,omitempty"`
	FromDate  time.Time `query:"from_date"`
	Include   bool      `query:"include_deleted"`
	SortBy    string    `json:"sort_by"`
	Mandatory string    `query:"mandatory" validate:"required"`
}

type searchUsersBody struct {
	Query string `json:"query"`
}

type queryRequiredRules struct {
	OptionalPtr      *string `query:"optional_ptr"`
	RequiredByTag    string  `query:"required_by_tag" validate:"required"`
	RequiredPtrByTag *string `query:"required_ptr_by_tag,omitempty" validate:"required"`
	OptionalByTag    string  `query:"optional_by_tag,omitempty"`
}

type queryTagValues struct {
	Limit   int    `query:"limit" default:"10" example:"50"`
	Verbose bool   `query:"verbose" default:"false" example:"true"`
	Mode    string `query:"mode" default:"basic" example:"advanced"`
}

type queryEnumValues struct {
	Status userStatus `query:"status"`
}

type querySwaggerIgnore struct {
	Visible string `query:"visible"`
	Hidden  int    `query:"hidden" swaggerignore:"true"`
}

type queryExtensions struct {
	Tenant string `query:"tenant" extensions:"x-nullable,x-owner=platform,!x-omitempty"`
}

type queryReplaceSkip struct {
	Age  sql.NullInt64  `query:"age"`
	Name sql.NullString `query:"name"`
}

type paginatedUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type patchUserBody struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

type extraSwaggerModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type overriddenErrorModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type phasedSwaggerCustomizer struct {
	events *[]string
}

func (c phasedSwaggerCustomizer) PreCustomizeSwagger(ctx *SwaggerContext) {
	if c.events != nil {
		*c.events = append(*c.events, "pre")
	}
}

func (c phasedSwaggerCustomizer) CustomizeSwagger(ctx *SwaggerContext) {
	if c.events != nil {
		*c.events = append(*c.events, "run")
	}
}

func (c phasedSwaggerCustomizer) PostCustomizeSwagger(ctx *SwaggerContext) {
	if c.events != nil {
		*c.events = append(*c.events, "post")
	}
	if ctx != nil {
		ctx.SetOperationField("x-post-ran", true)
	}
}

type preOnlySwaggerCustomizer struct {
	events *[]string
}

func (c preOnlySwaggerCustomizer) PreCustomizeSwagger(ctx *SwaggerContext) {
	if c.events != nil {
		*c.events = append(*c.events, "pre-only")
	}
}

type postOnlySwaggerCustomizer struct {
	events *[]string
}

func (c postOnlySwaggerCustomizer) PostCustomizeSwagger(ctx *SwaggerContext) {
	if c.events != nil {
		*c.events = append(*c.events, "post-only")
	}
}

func TestBuildOpenAPISpec_OmitsFallbackDescriptionWithoutMetadata(t *testing.T) {
	GetGlobalRegistry().Clear()

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method:  "GET",
			Path:    "/users",
			Handler: "func(fiber.Ctx) error",
		},
	}, Config{Name: "Test API"})

	getOp, ok := spec.Paths["/users"]["get"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GET /users operation")
	}

	if _, hasDescription := getOp["description"]; hasDescription {
		t.Fatalf("expected no fallback description, got %v", getOp["description"])
	}
}

func TestBuildOpenAPISpec_UsesMetadataDescription(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users/:id", &RouteMetadata{
		Summary:     "Get user",
		Description: "Returns a user by ID",
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method:  "GET",
			Path:    "/users/:id",
			Handler: "func(fiber.Ctx) error",
		},
	}, Config{Name: "Test API"})

	getOp, ok := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GET /users/{id} operation")
	}

	description, ok := getOp["description"].(string)
	if !ok {
		t.Fatalf("expected string description")
	}

	if description != "Returns a user by ID" {
		t.Fatalf("unexpected description: %q", description)
	}
}

func TestBuildOpenAPISpec_GeneratesQueryAndPathParameters(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users/:id", &RouteMetadata{
		QueryType: reflect.TypeOf(listUsersQuery{}),
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method: "GET",
			Path:   "/users/:id",
		},
	}, Config{Name: "Test API"})

	getOp, ok := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GET /users/{id} operation")
	}

	params, ok := getOp["parameters"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected parameters to be []map[string]interface{}")
	}

	if len(params) != 7 {
		t.Fatalf("expected 7 parameters, got %d", len(params))
	}

	byKey := map[string]map[string]interface{}{}
	for _, p := range params {
		key := p["in"].(string) + ":" + p["name"].(string)
		byKey[key] = p
	}

	if _, ok := byKey["path:id"]; !ok {
		t.Fatalf("expected path parameter id")
	}

	if byKey["query:page"]["required"] != false {
		t.Fatalf("expected query:page to be optional")
	}

	pageSchema := byKey["query:page"]["schema"].(map[string]interface{})
	if pageSchema["type"] != "integer" {
		t.Fatalf("expected query:page schema type integer, got %v", pageSchema["type"])
	}

	if byKey["query:search"]["required"] != false {
		t.Fatalf("expected query:search to be optional")
	}

	fromDateSchema := byKey["query:from_date"]["schema"].(map[string]interface{})
	if fromDateSchema["type"] != "string" || fromDateSchema["format"] != "date-time" {
		t.Fatalf("expected query:from_date to be date-time string, got %v", fromDateSchema)
	}

	includeSchema := byKey["query:include_deleted"]["schema"].(map[string]interface{})
	if includeSchema["type"] != "boolean" {
		t.Fatalf("expected query:include_deleted schema type boolean, got %v", includeSchema["type"])
	}

	if byKey["query:mandatory"]["required"] != true {
		t.Fatalf("expected query:mandatory to be required")
	}

	if _, ok := byKey["query:sort_by"]; !ok {
		t.Fatalf("expected fallback from json tag for sort_by")
	}
}

func TestBuildOpenAPISpec_QuerySwaggerIgnoreExcludesField(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{
		QueryType: reflect.TypeOf(querySwaggerIgnore{}),
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/users"]["get"].(map[string]interface{})
	params := getOp["parameters"].([]map[string]interface{})

	byName := map[string]map[string]interface{}{}
	for _, p := range params {
		byName[p["name"].(string)] = p
	}

	if _, ok := byName["visible"]; !ok {
		t.Fatalf("expected visible query param")
	}
	if _, ok := byName["hidden"]; ok {
		t.Fatalf("expected hidden query param to be excluded by swaggerignore")
	}
}

func TestBuildOpenAPISpec_QueryExtensionsAppliedToSchema(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{
		QueryType: reflect.TypeOf(queryExtensions{}),
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/users"]["get"].(map[string]interface{})
	params := getOp["parameters"].([]map[string]interface{})
	if len(params) != 1 {
		t.Fatalf("expected one query parameter, got %d", len(params))
	}

	schema := params[0]["schema"].(map[string]interface{})
	if schema["x-nullable"] != true {
		t.Fatalf("expected x-nullable=true")
	}
	if schema["x-owner"] != "platform" {
		t.Fatalf("expected x-owner=platform, got %v", schema["x-owner"])
	}
	if schema["x-omitempty"] != false {
		t.Fatalf("expected x-omitempty=false")
	}
}

func TestBuildOpenAPISpec_QueryTypeRulesReplaceAndSkip(t *testing.T) {
	ClearTypeRules()
	defer ClearTypeRules()
	ReplaceType[sql.NullInt64, int64]()
	SkipType[sql.NullString]()

	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{
		QueryType: reflect.TypeOf(queryReplaceSkip{}),
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/users"]["get"].(map[string]interface{})
	params := getOp["parameters"].([]map[string]interface{})

	if len(params) != 1 {
		t.Fatalf("expected only one query parameter after skip rule, got %d", len(params))
	}
	if params[0]["name"] != "age" {
		t.Fatalf("expected remaining query param to be age, got %v", params[0]["name"])
	}
	schema := params[0]["schema"].(map[string]interface{})
	if schema["type"] != "integer" {
		t.Fatalf("expected age schema type integer via replace rule, got %v", schema["type"])
	}
}

func TestBuildOpenAPISpecWithOptions_IncludesExtraModelsInComponents(t *testing.T) {
	GetGlobalRegistry().Clear()

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/health"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		ExtraModels: []reflect.Type{
			reflect.TypeOf(extraSwaggerModel{}),
		},
	})

	if spec.Components == nil {
		t.Fatalf("expected components to be initialized")
	}

	if _, ok := spec.Components.Schemas["extraSwaggerModel"]; !ok {
		t.Fatalf("expected extraSwaggerModel schema to be included in components")
	}
}

func TestWithSwaggerExtraModels_NormalizesAndSkipsInvalid(t *testing.T) {
	cfg := swaggerOptions{}

	WithSwaggerExtraModels(
		extraSwaggerModel{},
		reflect.TypeOf(&patchUserBody{}),
		nil,
	)(&cfg)

	if len(cfg.extraModels) != 2 {
		t.Fatalf("expected 2 extra models, got %d", len(cfg.extraModels))
	}
	if cfg.extraModels[0] != reflect.TypeOf(extraSwaggerModel{}) {
		t.Fatalf("unexpected first type: %v", cfg.extraModels[0])
	}
	if cfg.extraModels[1] != reflect.TypeOf(patchUserBody{}) {
		t.Fatalf("expected pointer type to be normalized to value type, got %v", cfg.extraModels[1])
	}
}

func TestCombineExtraModelTypes_Deduplicates(t *testing.T) {
	registry := NewSwaggerModelRegistry()
	registry.Add(extraSwaggerModel{})

	combined := combineExtraModelTypes(registry, []reflect.Type{
		reflect.TypeOf(extraSwaggerModel{}),
		reflect.TypeOf(patchUserBody{}),
	})

	if len(combined) != 2 {
		t.Fatalf("expected 2 unique models, got %d", len(combined))
	}
}

func TestSwaggerModelRegistry_AddNamed_SupportsAnonymousStructModel(t *testing.T) {
	ClearSchemaNameRegistry()
	defer ClearSchemaNameRegistry()
	GetGlobalRegistry().Clear()

	models := NewSwaggerModelRegistry()
	models.AddNamed("InlineUser", struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{})

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/health"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		ExtraModels: models.Types(),
	})

	if _, ok := spec.Components.Schemas["InlineUser"]; !ok {
		t.Fatalf("expected InlineUser schema to be included")
	}
}

func TestBuildOpenAPISpec_GeneratesTypedResponseContentAndNoContent(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/download", &RouteMetadata{
		Responses: map[int]ResponseMetadata{
			200: {
				Type:        reflect.TypeOf(""),
				ContentType: "text/plain",
			},
			204: {
				Type:        reflect.TypeOf(""),
				ContentType: "text/plain",
			},
			206: {
				ContentType: "application/octet-stream",
			},
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method: "GET",
			Path:   "/download",
		},
	}, Config{Name: "Test API"})

	getOp, ok := spec.Paths["/download"]["get"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GET /download operation")
	}

	responses, ok := getOp["responses"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected responses map")
	}

	resp200 := responses["200"].(map[string]interface{})
	content200 := resp200["content"].(map[string]interface{})
	textPlain := content200["text/plain"].(map[string]interface{})
	schema200 := textPlain["schema"].(map[string]interface{})
	if schema200["type"] != "string" {
		t.Fatalf("expected 200 text/plain schema to be string, got %v", schema200["type"])
	}

	resp204 := responses["204"].(map[string]interface{})
	if _, hasContent := resp204["content"]; hasContent {
		t.Fatalf("expected 204 response to omit content")
	}

	resp206 := responses["206"].(map[string]interface{})
	content206 := resp206["content"].(map[string]interface{})
	octetStream := content206["application/octet-stream"].(map[string]interface{})
	schema206 := octetStream["schema"].(map[string]interface{})
	if schema206["type"] != "string" || schema206["format"] != "binary" {
		t.Fatalf("expected 206 binary schema, got %v", schema206)
	}
}

func TestBuildOpenAPISpec_UsesCustomResponseDescription(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/custom-response-desc", &RouteMetadata{
		Responses: map[int]ResponseMetadata{
			200: {
				Type:        reflect.TypeOf(map[string]string{}),
				ContentType: "application/json",
				Description: "Successful user lookup",
			},
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/custom-response-desc"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/custom-response-desc"]["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})

	if resp200["description"] != "Successful user lookup" {
		t.Fatalf("expected custom response description, got %v", resp200["description"])
	}
}

func TestBuildOpenAPISpec_GeneratesResponseHeaders(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/response-headers", &RouteMetadata{
		Responses: map[int]ResponseMetadata{
			200: {
				Type:        reflect.TypeOf(map[string]string{}),
				ContentType: "application/json",
				Headers: map[string]ResponseHeaderMetadata{
					"X-Request-ID": {
						Type:        reflect.TypeOf(""),
						Description: "Request correlation ID",
					},
					"Set-Cookie": {
						Type:        reflect.TypeOf(""),
						Description: "Set-Cookie for session_id",
					},
				},
			},
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/response-headers"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/response-headers"]["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})
	headers := resp200["headers"].(map[string]interface{})

	xReqID := headers["X-Request-ID"].(map[string]interface{})
	if xReqID["description"] != "Request correlation ID" {
		t.Fatalf("unexpected X-Request-ID description: %v", xReqID["description"])
	}
	xReqIDSchema := xReqID["schema"].(map[string]interface{})
	if xReqIDSchema["type"] != "string" {
		t.Fatalf("expected X-Request-ID schema type string, got %v", xReqIDSchema["type"])
	}
}

func TestBuildOpenAPISpec_GeneratesOptionalMultiContentRequestBody(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("POST", "/users/search", &RouteMetadata{
		RequestBody: &RequestBodyMetadata{
			Type:         reflect.TypeOf(searchUsersBody{}),
			Required:     false,
			ContentTypes: []string{"application/json", "application/xml"},
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method: "POST",
			Path:   "/users/search",
		},
	}, Config{Name: "Test API"})

	postOp, ok := spec.Paths["/users/search"]["post"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected POST /users/search operation")
	}

	requestBody, ok := postOp["requestBody"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected requestBody map")
	}

	if requestBody["required"] != false {
		t.Fatalf("expected requestBody.required=false")
	}

	content, ok := requestBody["content"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected requestBody.content map")
	}

	if _, ok := content["application/json"]; !ok {
		t.Fatalf("expected application/json content")
	}
	if _, ok := content["application/xml"]; !ok {
		t.Fatalf("expected application/xml content")
	}
}

func TestBuildOpenAPISpec_QueryRequiredInferenceRules(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/rules", &RouteMetadata{
		QueryType: reflect.TypeOf(queryRequiredRules{}),
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method: "GET",
			Path:   "/rules",
		},
	}, Config{Name: "Test API"})

	getOp, ok := spec.Paths["/rules"]["get"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GET /rules operation")
	}

	params, ok := getOp["parameters"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected parameters to be []map[string]interface{}")
	}

	byName := map[string]map[string]interface{}{}
	for _, p := range params {
		if in, _ := p["in"].(string); in == "query" {
			byName[p["name"].(string)] = p
		}
	}

	if byName["optional_ptr"]["required"] != false {
		t.Fatalf("expected optional_ptr to be optional")
	}
	if byName["required_by_tag"]["required"] != true {
		t.Fatalf("expected required_by_tag to be required")
	}
	if byName["required_ptr_by_tag"]["required"] != true {
		t.Fatalf("expected required_ptr_by_tag to be required via validate tag")
	}
	if byName["optional_by_tag"]["required"] != false {
		t.Fatalf("expected optional_by_tag to be optional")
	}
}

func TestBuildOpenAPISpec_GeneratesHeaderAndCookieParameters(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/me/:id", &RouteMetadata{
		Parameters: []ParameterMetadata{
			{
				Name:       "X-Tenant-ID",
				In:         "header",
				Type:       reflect.TypeOf(""),
				Required:   true,
				Extensions: "x-nullable,x-owner=platform,!x-omitempty",
			},
			{
				Name:       "session_id",
				In:         "cookie",
				Type:       reflect.TypeOf(0),
				Required:   false,
				Extensions: "x-format=legacy",
			},
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method: "GET",
			Path:   "/me/:id",
		},
	}, Config{Name: "Test API"})

	getOp, ok := spec.Paths["/me/{id}"]["get"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GET /me/{id} operation")
	}

	params, ok := getOp["parameters"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected parameters to be []map[string]interface{}")
	}

	byKey := map[string]map[string]interface{}{}
	for _, p := range params {
		key := p["in"].(string) + ":" + p["name"].(string)
		byKey[key] = p
	}

	if _, ok := byKey["path:id"]; !ok {
		t.Fatalf("expected path:id parameter")
	}

	header := byKey["header:X-Tenant-ID"]
	if header["required"] != true {
		t.Fatalf("expected required header parameter")
	}
	headerSchema := header["schema"].(map[string]interface{})
	if headerSchema["type"] != "string" {
		t.Fatalf("expected header schema type string, got %v", headerSchema["type"])
	}
	if headerSchema["x-nullable"] != true || headerSchema["x-owner"] != "platform" || headerSchema["x-omitempty"] != false {
		t.Fatalf("unexpected header extensions: %v", headerSchema)
	}

	cookie := byKey["cookie:session_id"]
	if cookie["required"] != false {
		t.Fatalf("expected optional cookie parameter")
	}
	cookieSchema := cookie["schema"].(map[string]interface{})
	if cookieSchema["type"] != "integer" {
		t.Fatalf("expected cookie schema type integer, got %v", cookieSchema["type"])
	}
	if cookieSchema["x-format"] != "legacy" {
		t.Fatalf("expected cookie extension x-format=legacy, got %v", cookieSchema["x-format"])
	}
}

func TestBuildOpenAPISpec_QueryParamsIncludeExampleAndDefaultTags(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/search", &RouteMetadata{
		QueryType: reflect.TypeOf(queryTagValues{}),
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/search"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/search"]["get"].(map[string]interface{})
	params := getOp["parameters"].([]map[string]interface{})

	byName := map[string]map[string]interface{}{}
	for _, p := range params {
		if p["in"] == "query" {
			byName[p["name"].(string)] = p
		}
	}

	limitSchema := byName["limit"]["schema"].(map[string]interface{})
	if limitSchema["default"] != int64(10) || limitSchema["example"] != int64(50) {
		t.Fatalf("unexpected limit query schema values: %v", limitSchema)
	}

	verboseSchema := byName["verbose"]["schema"].(map[string]interface{})
	if verboseSchema["default"] != false || verboseSchema["example"] != true {
		t.Fatalf("unexpected verbose query schema values: %v", verboseSchema)
	}

	modeSchema := byName["mode"]["schema"].(map[string]interface{})
	if modeSchema["default"] != "basic" || modeSchema["example"] != "advanced" {
		t.Fatalf("unexpected mode query schema values: %v", modeSchema)
	}
}

func TestBuildOpenAPISpec_QueryParamsIncludeRegisteredEnum(t *testing.T) {
	ClearEnumRegistry()
	defer ClearEnumRegistry()
	RegisterEnum(statusActive, statusPending)

	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/enum-search", &RouteMetadata{
		QueryType: reflect.TypeOf(queryEnumValues{}),
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/enum-search"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/enum-search"]["get"].(map[string]interface{})
	params := getOp["parameters"].([]map[string]interface{})
	var statusParam map[string]interface{}
	for _, p := range params {
		if p["in"] == "query" && p["name"] == "status" {
			statusParam = p
			break
		}
	}
	if statusParam == nil {
		t.Fatalf("expected status query parameter")
	}
	schema := statusParam["schema"].(map[string]interface{})
	enumValues := schema["enum"].([]interface{})
	if len(enumValues) != 2 || enumValues[0] != statusActive || enumValues[1] != statusPending {
		t.Fatalf("unexpected query enum values: %v", enumValues)
	}
}

func TestBuildOpenAPISpec_PaginatedAddsParamsAndEnvelope(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{
		Pagination: &PaginationMetadata{
			ItemType: reflect.TypeOf(paginatedUser{}),
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/users"]["get"].(map[string]interface{})

	params := getOp["parameters"].([]map[string]interface{})
	byName := map[string]map[string]interface{}{}
	for _, p := range params {
		byName[p["in"].(string)+":"+p["name"].(string)] = p
	}
	for _, key := range []string{"query:page", "query:limit", "query:sort", "query:cursor"} {
		if _, ok := byName[key]; !ok {
			t.Fatalf("expected pagination parameter %s", key)
		}
	}

	responses := getOp["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})
	content := resp200["content"].(map[string]interface{})
	jsonContent := content["application/json"].(map[string]interface{})
	schema := jsonContent["schema"].(map[string]interface{})
	props := schema["properties"].(map[string]interface{})
	items := props["items"].(map[string]interface{})
	if items["type"] != "array" {
		t.Fatalf("expected paginated items to be array, got %v", items["type"])
	}
}

func TestBuildOpenAPISpec_PaginatedDoesNotOverrideExplicit200(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{
		Pagination: &PaginationMetadata{
			ItemType: reflect.TypeOf(paginatedUser{}),
		},
		Responses: map[int]ResponseMetadata{
			200: {
				Type:        reflect.TypeOf(map[string]string{}),
				ContentType: "application/json",
			},
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users"},
	}, Config{Name: "Test API"})

	getOp := spec.Paths["/users"]["get"].(map[string]interface{})
	responses := getOp["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})
	content := resp200["content"].(map[string]interface{})
	jsonContent := content["application/json"].(map[string]interface{})
	schema := jsonContent["schema"].(map[string]interface{})

	if _, hasPaginatedProps := schema["properties"]; hasPaginatedProps {
		t.Fatalf("expected explicit 200 response schema to be preserved (not paginated envelope), got %v", schema)
	}
}

func TestBuildOpenAPISpec_BodyAtLeastOneGeneratesAnyOfRequiredConstraint(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("PATCH", "/users/:id", &RouteMetadata{
		RequestBody: &RequestBodyMetadata{
			Type:              reflect.TypeOf(patchUserBody{}),
			Required:          true,
			ContentTypes:      []string{"application/json"},
			RequireAtLeastOne: true,
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "PATCH", Path: "/users/:id"},
	}, Config{Name: "Test API"})

	patchOp := spec.Paths["/users/{id}"]["patch"].(map[string]interface{})
	requestBody := patchOp["requestBody"].(map[string]interface{})
	content := requestBody["content"].(map[string]interface{})
	jsonContent := content["application/json"].(map[string]interface{})
	schema := jsonContent["schema"].(map[string]interface{})

	anyOf, ok := schema["anyOf"].([]map[string]interface{})
	if !ok {
		// For referenced schemas, we emit allOf: [$ref, {anyOf: ...}]
		allOf, hasAllOf := schema["allOf"].([]map[string]interface{})
		if !hasAllOf || len(allOf) != 2 {
			t.Fatalf("expected direct anyOf or allOf wrapper for at-least-one-field body, got %v", schema)
		}
		innerAnyOf, innerOK := allOf[1]["anyOf"].([]map[string]interface{})
		if !innerOK {
			t.Fatalf("expected anyOf in allOf wrapper, got %v", allOf[1])
		}
		anyOf = innerAnyOf
	}
	if len(anyOf) != 2 {
		t.Fatalf("expected anyOf with two entries, got %d", len(anyOf))
	}
}

func TestBuildOpenAPISpecWithSecuritySchemes_AddsGlobalComponentsSecuritySchemes(t *testing.T) {
	GetGlobalRegistry().Clear()

	securitySchemes := map[string]interface{}{
		"BearerAuth": map[string]interface{}{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
		},
		"ApiKeyAuth": map[string]interface{}{
			"type": "apiKey",
			"name": "X-API-Key",
			"in":   "header",
		},
	}

	spec := BuildOpenAPISpecWithSecuritySchemes([]RouteInfo{
		{
			Method: "GET",
			Path:   "/health",
		},
	}, Config{Name: "Test API"}, securitySchemes)

	if spec.Components == nil {
		t.Fatalf("expected components")
	}

	if len(spec.Components.SecuritySchemes) != 2 {
		t.Fatalf("expected 2 security schemes, got %d", len(spec.Components.SecuritySchemes))
	}

	bearer := spec.Components.SecuritySchemes["BearerAuth"].(map[string]interface{})
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Fatalf("unexpected bearer scheme: %v", bearer)
	}

	apiKey := spec.Components.SecuritySchemes["ApiKeyAuth"].(map[string]interface{})
	if apiKey["type"] != "apiKey" || apiKey["in"] != "header" || apiKey["name"] != "X-API-Key" {
		t.Fatalf("unexpected api key scheme: %v", apiKey)
	}
}

func TestBuildOpenAPISpec_GeneratesOperationSecurityRequirements(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/private", &RouteMetadata{
		Security: []SecurityRequirement{
			{
				Scheme: "BearerAuth",
			},
			{
				Scheme: "OAuth2",
				Scopes: []string{"read:users", "write:users"},
			},
		},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{
			Method: "GET",
			Path:   "/private",
		},
	}, Config{Name: "Test API"})

	getOp, ok := spec.Paths["/private"]["get"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected GET /private operation")
	}

	security, ok := getOp["security"].([]map[string][]string)
	if !ok {
		t.Fatalf("expected security array")
	}

	if len(security) != 2 {
		t.Fatalf("expected 2 security requirements, got %d", len(security))
	}

	if scopes := security[0]["BearerAuth"]; len(scopes) != 0 {
		t.Fatalf("expected empty scopes for bearer auth, got %v", scopes)
	}

	oauthScopes := security[1]["OAuth2"]
	if len(oauthScopes) != 2 || oauthScopes[0] != "read:users" || oauthScopes[1] != "write:users" {
		t.Fatalf("unexpected oauth scopes: %v", oauthScopes)
	}
}

func TestBuildOpenAPISpecWithSecurity_DefaultAndPublicOverride(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/public", &RouteMetadata{
		Security: []SecurityRequirement{},
	})
	GetGlobalRegistry().RegisterRoute("GET", "/override", &RouteMetadata{
		Security: []SecurityRequirement{
			{
				Scheme: "ApiKeyAuth",
			},
		},
	})

	spec := BuildOpenAPISpecWithSecurity([]RouteInfo{
		{Method: "GET", Path: "/private"},
		{Method: "GET", Path: "/public"},
		{Method: "GET", Path: "/override"},
	}, Config{Name: "Test API"}, map[string]interface{}{
		"BearerAuth": map[string]interface{}{
			"type":   "http",
			"scheme": "bearer",
		},
		"ApiKeyAuth": map[string]interface{}{
			"type": "apiKey",
			"name": "X-API-Key",
			"in":   "header",
		},
	}, []SecurityRequirement{
		{
			Scheme: "BearerAuth",
		},
	})

	if len(spec.Security) != 1 {
		t.Fatalf("expected one global security requirement, got %d", len(spec.Security))
	}
	if _, ok := spec.Security[0]["BearerAuth"]; !ok {
		t.Fatalf("expected global BearerAuth security requirement")
	}

	privateGet := spec.Paths["/private"]["get"].(map[string]interface{})
	if _, hasSecurity := privateGet["security"]; hasSecurity {
		t.Fatalf("expected private operation to inherit global security without operation override")
	}

	publicGet := spec.Paths["/public"]["get"].(map[string]interface{})
	publicSecurity, ok := publicGet["security"].([]map[string][]string)
	if !ok {
		t.Fatalf("expected public operation to have explicit empty security")
	}
	if len(publicSecurity) != 0 {
		t.Fatalf("expected public operation security to be empty, got %v", publicSecurity)
	}

	overrideGet := spec.Paths["/override"]["get"].(map[string]interface{})
	overrideSecurity, ok := overrideGet["security"].([]map[string][]string)
	if !ok {
		t.Fatalf("expected override operation security array")
	}
	if len(overrideSecurity) != 1 {
		t.Fatalf("expected one override security requirement, got %d", len(overrideSecurity))
	}
	if _, ok := overrideSecurity[0]["ApiKeyAuth"]; !ok {
		t.Fatalf("expected ApiKeyAuth override security requirement")
	}
}

func TestBuildOpenAPISpecWithOptions_AddsInfoMetadataAndTagDescriptions(t *testing.T) {
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{
		Tags: []string{"Users"},
	})
	GetGlobalRegistry().RegisterRoute("GET", "/health", &RouteMetadata{
		Tags: []string{"System"},
	})

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/users"},
		{Method: "GET", Path: "/health"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		TermsOfService: "https://example.com/terms",
		Contact: &AutoSwaggerContact{
			Name:  "API Team",
			URL:   "https://example.com/contact",
			Email: "api@example.com",
		},
		License: &AutoSwaggerLicense{
			Name: "MIT",
			URL:  "https://opensource.org/license/mit",
		},
		TagDescriptions: map[string]string{
			"Users":  "User operations",
			"System": "System endpoints",
		},
	})

	if spec.Info.TermsOfService != "https://example.com/terms" {
		t.Fatalf("unexpected termsOfService: %q", spec.Info.TermsOfService)
	}
	if spec.Info.Contact == nil || spec.Info.Contact.Email != "api@example.com" {
		t.Fatalf("expected contact metadata, got %+v", spec.Info.Contact)
	}
	if spec.Info.License == nil || spec.Info.License.Name != "MIT" {
		t.Fatalf("expected license metadata, got %+v", spec.Info.License)
	}

	if len(spec.Tags) != 2 {
		t.Fatalf("expected two top-level tags, got %d", len(spec.Tags))
	}

	byName := map[string]AutoSwaggerTag{}
	for _, tag := range spec.Tags {
		byName[tag.Name] = tag
	}

	if byName["Users"].Description != "User operations" {
		t.Fatalf("unexpected Users description: %q", byName["Users"].Description)
	}
	if byName["System"].Description != "System endpoints" {
		t.Fatalf("unexpected System description: %q", byName["System"].Description)
	}
}

func TestBuildOpenAPISpec_GeneratesDeterministicOperationIDs(t *testing.T) {
	ClearOperationIDHook()
	defer ClearOperationIDHook()
	GetGlobalRegistry().Clear()

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users"},
		{Method: "GET", Path: "/users/:id"},
		{Method: "POST", Path: "/users"},
		{Method: "PATCH", Path: "/users/:id"},
	}, Config{Name: "Test API"})

	getUsers := spec.Paths["/users"]["get"].(map[string]interface{})
	if getUsers["operationId"] != "listUser" {
		t.Fatalf("unexpected operationId for GET /users: %v", getUsers["operationId"])
	}

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "getUserById" {
		t.Fatalf("unexpected operationId for GET /users/{id}: %v", getUser["operationId"])
	}

	postUser := spec.Paths["/users"]["post"].(map[string]interface{})
	if postUser["operationId"] != "createUser" {
		t.Fatalf("unexpected operationId for POST /users: %v", postUser["operationId"])
	}

	patchUser := spec.Paths["/users/{id}"]["patch"].(map[string]interface{})
	if patchUser["operationId"] != "patchUserById" {
		t.Fatalf("unexpected operationId for PATCH /users/{id}: %v", patchUser["operationId"])
	}
}

func TestBuildOpenAPISpec_OperationIDUniquenessWithSuffix(t *testing.T) {
	ClearOperationIDHook()
	defer ClearOperationIDHook()
	GetGlobalRegistry().Clear()
	GetGlobalRegistry().RegisterRoute("GET", "/users", &RouteMetadata{OperationID: "customID"})
	GetGlobalRegistry().RegisterRoute("GET", "/admins", &RouteMetadata{OperationID: "customID"})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users"},
		{Method: "GET", Path: "/admins"},
	}, Config{Name: "Test API"})

	getUsers := spec.Paths["/users"]["get"].(map[string]interface{})
	getAdmins := spec.Paths["/admins"]["get"].(map[string]interface{})

	if getUsers["operationId"] != "customID" {
		t.Fatalf("unexpected first operationId: %v", getUsers["operationId"])
	}
	if getAdmins["operationId"] != "customID_2" {
		t.Fatalf("expected suffixed operationId, got %v", getAdmins["operationId"])
	}
}

func TestBuildOpenAPISpec_GlobalOperationIDTagPrefixForExplicitName(t *testing.T) {
	ClearOperationIDHook()
	defer ClearOperationIDHook()
	ClearOperationIDTagPrefix()
	defer ClearOperationIDTagPrefix()
	GetGlobalRegistry().Clear()

	RegisterOperationIDTagPrefix("_")
	GetGlobalRegistry().RegisterRoute("GET", "/users/:id", &RouteMetadata{
		OperationID: "GetUserByID",
		Tags:        []string{"Users"},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "Users_GetUserByID" {
		t.Fatalf("expected prefixed operationId, got %v", getUser["operationId"])
	}
}

func TestBuildOpenAPISpec_GlobalOperationIDTagPrefixForGeneratedName(t *testing.T) {
	ClearOperationIDHook()
	defer ClearOperationIDHook()
	ClearOperationIDTagPrefix()
	defer ClearOperationIDTagPrefix()
	GetGlobalRegistry().Clear()

	RegisterOperationIDTagPrefix("_")
	GetGlobalRegistry().RegisterRoute("GET", "/users/:id", &RouteMetadata{
		Tags: []string{"Users"},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "Users_getUserById" {
		t.Fatalf("expected prefixed generated operationId, got %v", getUser["operationId"])
	}
}

func TestBuildOpenAPISpec_OperationIDHookCanCustomizeFromContext(t *testing.T) {
	ClearOperationIDHook()
	defer ClearOperationIDHook()
	ClearOperationIDTagPrefix()
	defer ClearOperationIDTagPrefix()
	GetGlobalRegistry().Clear()

	RegisterOperationIDHook(func(ctx OperationIDHookContext) string {
		tag := "General"
		if len(ctx.Tags) > 0 {
			tag = ctx.Tags[0]
		}
		return tag + "__" + ctx.OperationID
	})

	GetGlobalRegistry().RegisterRoute("GET", "/users/:id", &RouteMetadata{
		OperationID: "GetUserByID",
		Tags:        []string{"Users"},
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "Users__GetUserByID" {
		t.Fatalf("expected hook-customized operationId, got %v", getUser["operationId"])
	}
}

func TestBuildOpenAPISpec_OperationIDHookAppliesToGeneratedIDs(t *testing.T) {
	ClearOperationIDHook()
	defer ClearOperationIDHook()
	ClearOperationIDTagPrefix()
	defer ClearOperationIDTagPrefix()
	GetGlobalRegistry().Clear()

	RegisterOperationIDHook(func(ctx OperationIDHookContext) string {
		if ctx.Generated {
			return "Auto_" + ctx.OperationID
		}
		return ctx.OperationID
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "Auto_getUserById" {
		t.Fatalf("expected generated operationId to be customized by hook, got %v", getUser["operationId"])
	}
}

func TestBuildOpenAPISpec_RegisterHookCanMutateAnyMetadata(t *testing.T) {
	ClearGlobalHook()
	defer ClearGlobalHook()
	ClearOperationIDHook()
	defer ClearOperationIDHook()
	ClearOperationIDTagPrefix()
	defer ClearOperationIDTagPrefix()
	GetGlobalRegistry().Clear()

	RegisterGlobalHook(func(ctx *SwaggerContext) {
		if ctx.Route.Path != "/users/:id" || strings.ToUpper(ctx.Route.Method) != "GET" {
			return
		}
		ctx.Metadata.OperationID = "Users_GetUserByID"
		ctx.Metadata.Tags = []string{"Users"}
		ctx.Metadata.Summary = "Hooked summary"
		ctx.Metadata.Description = "Hooked description"
		ctx.Metadata.Parameters = append(ctx.Metadata.Parameters, ParameterMetadata{
			Name:     "X-Tenant-ID",
			In:       "header",
			Type:     reflect.TypeOf(""),
			Required: true,
		})
		ctx.Metadata.Responses = map[int]ResponseMetadata{
			200: {
				Type:        reflect.TypeOf(map[string]string{}),
				ContentType: "application/json",
				Description: "Hooked OK",
			},
		}
	})

	spec := BuildOpenAPISpec([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "Users_GetUserByID" {
		t.Fatalf("expected hook operationId, got %v", getUser["operationId"])
	}
	if getUser["summary"] != "Hooked summary" {
		t.Fatalf("expected hook summary, got %v", getUser["summary"])
	}
	if getUser["description"] != "Hooked description" {
		t.Fatalf("expected hook description, got %v", getUser["description"])
	}

	params := getUser["parameters"].([]map[string]interface{})
	foundHeader := false
	for _, p := range params {
		if p["in"] == "header" && p["name"] == "X-Tenant-ID" {
			foundHeader = true
			break
		}
	}
	if !foundHeader {
		t.Fatalf("expected header parameter injected by hook")
	}

	responses := getUser["responses"].(map[string]interface{})
	resp200 := responses["200"].(map[string]interface{})
	if resp200["description"] != "Hooked OK" {
		t.Fatalf("expected custom 200 description from hook, got %v", resp200["description"])
	}
}

func TestBuildOpenAPISpecWithOptions_HookOverridesGlobalRegisterHook(t *testing.T) {
	ClearGlobalHook()
	defer ClearGlobalHook()
	GetGlobalRegistry().Clear()

	RegisterGlobalHook(func(ctx *SwaggerContext) {
		ctx.Metadata.OperationID = "FromGlobalHook"
	})

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		Hook: func(ctx *SwaggerContext) {
			ctx.Metadata.OperationID = "FromOptionHook"
			ctx.Metadata.Tags = []string{"Users"}
		},
	})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "FromOptionHook" {
		t.Fatalf("expected option hook to override global hook, got %v", getUser["operationId"])
	}
}

func TestBuildOpenAPISpec_SkipsDuplicateMethodPathRoutesWithoutOperationIDSuffix(t *testing.T) {
	ClearGlobalHook()
	defer ClearGlobalHook()
	GetGlobalRegistry().Clear()

	GetGlobalRegistry().RegisterRoute("GET", "/health", &RouteMetadata{
		OperationID: "HealthCheck",
		Summary:     "Health check",
	})

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/health"},
		{Method: "GET", Path: "/health"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		Hook: func(ctx *SwaggerContext) {
			ctx.Metadata.OperationID = "custom_" + ctx.Metadata.OperationID
		},
	})

	getOp := spec.Paths["/health"]["get"].(map[string]interface{})
	if getOp["operationId"] != "custom_HealthCheck" {
		t.Fatalf("expected no operationId suffix for duplicate method/path, got %v", getOp["operationId"])
	}
}

func TestBuildOpenAPISpecWithOptions_HookCanAddNamedModelViaContext(t *testing.T) {
	ClearGlobalHook()
	defer ClearGlobalHook()
	ClearSchemaNameRegistry()
	defer ClearSchemaNameRegistry()
	GetGlobalRegistry().Clear()

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		Hook: func(ctx *SwaggerContext) {
			ctx.Metadata.Tags = []string{"Users"}
			ctx.Metadata.OperationID = "GetUserByID"
			ctx.Metadata.OperationID = ctx.Metadata.Tags[0] + "__" + ctx.Metadata.OperationID
			if ctx.Models != nil {
				ctx.Models.AddNamed("CustomTest", struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{})
			}
		},
	})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "Users__GetUserByID" {
		t.Fatalf("expected operationId to include hook prefix behavior, got %v", getUser["operationId"])
	}

	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Fatalf("expected components.schemas to be present")
	}
	if _, ok := spec.Components.Schemas["CustomTest"]; !ok {
		t.Fatalf("expected CustomTest schema to be added from hook context models")
	}
}

func TestBuildOpenAPISpecWithOptions_HookCanOverrideErrorResponseSchemaByName(t *testing.T) {
	ClearGlobalHook()
	defer ClearGlobalHook()
	ClearSchemaNameRegistry()
	defer ClearSchemaNameRegistry()
	GetGlobalRegistry().Clear()

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		Hook: func(ctx *SwaggerContext) {
			if ctx.Models != nil {
				ctx.Models.AddNamed("ErrorResponse", overriddenErrorModel{})
			}
		},
	})

	schema, ok := spec.Components.Schemas["ErrorResponse"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ErrorResponse schema map")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ErrorResponse properties map")
	}
	if _, hasID := props["id"]; !hasID {
		t.Fatalf("expected overridden ErrorResponse to contain id field")
	}
	if _, hasError := props["error"]; hasError {
		t.Fatalf("expected default error field to be replaced by overridden model")
	}
}

func TestWithSwaggerCustomize_ComposesMultipleHooks(t *testing.T) {
	GetGlobalRegistry().Clear()

	cfg := swaggerOptions{}
	WithSwaggerCustomize(func(ctx *SwaggerContext) {
		ctx.Metadata.Tags = []string{"Users"}
	})(&cfg)
	WithSwaggerCustomize(func(ctx *SwaggerContext) {
		ctx.Metadata.OperationID = ctx.Metadata.Tags[0] + "__GetUserByID"
	})(&cfg)

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		Hook: cfg.hook,
	})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["operationId"] != "Users__GetUserByID" {
		t.Fatalf("expected chained hooks to run in order, got %v", getUser["operationId"])
	}
}

func TestComposeSwaggerCustomizeHook_AppliesDICustomizersAfterBaseHook(t *testing.T) {
	ClearSchemaNameRegistry()
	defer ClearSchemaNameRegistry()

	ctx := &SwaggerContext{
		Route: RouteInfo{Method: "GET", Path: "/users/:id"},
		Metadata: &RouteMetadata{
			Tags:        []string{"Users"},
			OperationID: "GetUserByID",
		},
		Models: NewSwaggerModelRegistry(),
	}

	base := func(ctx *SwaggerContext) {
		ctx.Metadata.OperationID = "Base_" + ctx.Metadata.OperationID
	}
	customizers := []SwaggerCustomizer{
		SwaggerCustomizeFunc(func(ctx *SwaggerContext) {
			ctx.Metadata.OperationID = ctx.Metadata.Tags[0] + "__" + ctx.Metadata.OperationID
		}),
		SwaggerCustomizeFunc(func(ctx *SwaggerContext) {
			ctx.Models.AddNamed("CustomTest", struct {
				ID string `json:"id"`
			}{})
		}),
	}

	hooks := composeSwaggerCustomizeHooks(base, customizers, nil, nil)
	if hooks.run == nil {
		t.Fatalf("expected composed hook")
	}
	hooks.run(ctx)

	if ctx.Metadata.OperationID != "Users__Base_GetUserByID" {
		t.Fatalf("unexpected operationId: %s", ctx.Metadata.OperationID)
	}
	types := ctx.Models.Types()
	if len(types) != 1 {
		t.Fatalf("expected one model from customizer, got %d", len(types))
	}
	if name, ok := getRegisteredSchemaName(types[0]); !ok || name != "CustomTest" {
		t.Fatalf("expected renamed model CustomTest, got %q (ok=%v)", name, ok)
	}
}

func TestComposeSwaggerCustomizeHooks_RunsPreAndPostWhenImplemented(t *testing.T) {
	events := []string{}
	customizers := []SwaggerCustomizer{
		phasedSwaggerCustomizer{events: &events},
	}

	preCustomizers := []SwaggerPreRun{customizers[0].(SwaggerPreRun)}
	postCustomizers := []SwaggerPostRun{customizers[0].(SwaggerPostRun)}
	hooks := composeSwaggerCustomizeHooks(nil, customizers, preCustomizers, postCustomizers)
	ctx := &SwaggerContext{
		Operation: map[string]interface{}{},
	}
	if hooks.pre != nil {
		hooks.pre(ctx)
	}
	if hooks.run != nil {
		hooks.run(ctx)
	}
	if hooks.post != nil {
		hooks.post(ctx)
	}

	if len(events) != 3 || events[0] != "pre" || events[1] != "run" || events[2] != "post" {
		t.Fatalf("unexpected phase order: %v", events)
	}
	if ctx.Operation["x-post-ran"] != true {
		t.Fatalf("expected post phase to mutate operation")
	}
}

func TestComposeSwaggerCustomizeHooks_SupportsPreOnlyAndPostOnlyInterfaces(t *testing.T) {
	events := []string{}
	preCustomizers := []SwaggerPreRun{
		preOnlySwaggerCustomizer{events: &events},
	}
	postCustomizers := []SwaggerPostRun{
		postOnlySwaggerCustomizer{events: &events},
	}

	hooks := composeSwaggerCustomizeHooks(nil, nil, preCustomizers, postCustomizers)
	ctx := &SwaggerContext{}
	if hooks.pre != nil {
		hooks.pre(ctx)
	}
	if hooks.post != nil {
		hooks.post(ctx)
	}

	if len(events) != 2 || events[0] != "pre-only" || events[1] != "post-only" {
		t.Fatalf("unexpected pre/post-only order: %v", events)
	}
}

func TestSwaggerContext_HelpersAndOperationSpecCustomization(t *testing.T) {
	ClearSchemaNameRegistry()
	defer ClearSchemaNameRegistry()
	GetGlobalRegistry().Clear()

	spec := BuildOpenAPISpecWithOptions([]RouteInfo{
		{Method: "GET", Path: "/users/:id"},
	}, Config{Name: "Test API"}, OpenAPIBuildOptions{
		Hook: func(ctx *SwaggerContext) {
			if ctx.Method != "GET" {
				t.Fatalf("expected method GET, got %s", ctx.Method)
			}
			if ctx.Path != "/users/{id}" {
				t.Fatalf("expected normalized path /users/{id}, got %s", ctx.Path)
			}

			ctx.SetSummary("Custom summary")
			ctx.SetDescription("Custom description")
			ctx.SetOperationID("Users_GetUserByID")
			ctx.AddParameter(ParameterMetadata{
				Name:     "X-Request-ID",
				In:       "header",
				Type:     reflect.TypeOf(""),
				Required: true,
			})
			ctx.SetRequestBody(struct {
				Name string `json:"name"`
			}{}, true)
			ctx.SetResponse(418, struct {
				Message string `json:"message"`
			}{}, "Teapot")
			ctx.SetOperationField("deprecated", true)
			ctx.Spec.Info.Version = "2.0.0"
		},
	})

	getUser := spec.Paths["/users/{id}"]["get"].(map[string]interface{})
	if getUser["summary"] != "Custom summary" {
		t.Fatalf("unexpected summary: %v", getUser["summary"])
	}
	if getUser["description"] != "Custom description" {
		t.Fatalf("unexpected description: %v", getUser["description"])
	}
	if getUser["operationId"] != "Users_GetUserByID" {
		t.Fatalf("unexpected operationId: %v", getUser["operationId"])
	}
	if getUser["deprecated"] != true {
		t.Fatalf("expected deprecated=true on operation")
	}
	if _, ok := getUser["requestBody"]; !ok {
		t.Fatalf("expected requestBody from context helper")
	}

	params := getUser["parameters"].([]map[string]interface{})
	foundHeader := false
	for _, p := range params {
		if p["in"] == "header" && p["name"] == "X-Request-ID" {
			foundHeader = true
			break
		}
	}
	if !foundHeader {
		t.Fatalf("expected header parameter from context helper")
	}

	responses := getUser["responses"].(map[string]interface{})
	if _, ok := responses["418"]; !ok {
		t.Fatalf("expected 418 response from context helper")
	}

	if spec.Info.Version != "2.0.0" {
		t.Fatalf("expected spec version override from context, got %s", spec.Info.Version)
	}
}

func TestSwaggerContext_PrimaryModelPackageHelpers(t *testing.T) {
	ctx := &SwaggerContext{
		Metadata: &RouteMetadata{
			RequestBody: &RequestBodyMetadata{
				Type: reflect.TypeOf(searchUsersBody{}),
			},
			Responses: map[int]ResponseMetadata{
				200: {Type: reflect.TypeOf(paginatedUser{})},
			},
		},
	}

	primary := ctx.RouteModelType()
	if primary == nil {
		t.Fatalf("expected primary model type")
	}
	if primary.Name() != "searchUsersBody" {
		t.Fatalf("expected request body type priority, got %s", primary.Name())
	}

	pkgPath := ctx.RouteModelPackagePath()
	if pkgPath == "" {
		t.Fatalf("expected package path for primary model")
	}

	pkgName := ctx.RouteModelPackageName()
	if pkgName == "" {
		t.Fatalf("expected package name for primary model")
	}
}

func TestSwaggerContext_PrimaryModelTypeFallsBackToLowestResponseCode(t *testing.T) {
	ctx := &SwaggerContext{
		Metadata: &RouteMetadata{
			Responses: map[int]ResponseMetadata{
				500: {Type: reflect.TypeOf(queryExtensions{})},
				200: {Type: reflect.TypeOf(paginatedUser{})},
			},
		},
	}

	primary := ctx.RouteModelType()
	if primary == nil {
		t.Fatalf("expected primary model type")
	}
	if primary.Name() != "paginatedUser" {
		t.Fatalf("expected lowest-status response model paginatedUser, got %s", primary.Name())
	}
}
