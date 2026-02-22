package web

import (
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v3"
)

var nonWordOperationIDChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// RouteBuilder provides fluent API for route metadata configuration
type RouteBuilder struct {
	method   string
	path     string
	router   fiber.Router
	registry *MetadataRegistry
	metadata *RouteMetadata
}

// RouteOption applies reusable configuration to a RouteBuilder.
type RouteOption func(*RouteBuilder) *RouteBuilder

// newRouteBuilder creates a new route builder
func newRouteBuilder(method, path string, router fiber.Router, registry *MetadataRegistry, inheritedTags []string, handlers []fiber.Handler) *RouteBuilder {
	// Fiber v3 API signature: (path, handler, ...handlers)
	// We need at least one handler
	if len(handlers) == 0 {
		// If no handlers provided, register a dummy route
		// The route's actual handler will be defined elsewhere
		handlers = append(handlers, func(c fiber.Ctx) error {
			return c.Next()
		})
	}

	// Register the route with Fiber
	// Fiber v3 requires: path string, handler any, ...middleware
	firstHandler := any(handlers[0])
	restHandlers := make([]any, len(handlers)-1)
	for i := 1; i < len(handlers); i++ {
		restHandlers[i-1] = handlers[i]
	}

	switch strings.ToUpper(method) {
	case "GET":
		router.Get(path, firstHandler, restHandlers...)
	case "POST":
		router.Post(path, firstHandler, restHandlers...)
	case "PUT":
		router.Put(path, firstHandler, restHandlers...)
	case "DELETE":
		router.Delete(path, firstHandler, restHandlers...)
	case "PATCH":
		router.Patch(path, firstHandler, restHandlers...)
	case "HEAD":
		router.Head(path, firstHandler, restHandlers...)
	case "OPTIONS":
		router.Options(path, firstHandler, restHandlers...)
	case "ALL":
		router.All(path, firstHandler, restHandlers...)
	}

	fullPath := resolveRegisteredPath(router, path)

	// Copy inherited tags
	tags := make([]string, len(inheritedTags))
	copy(tags, inheritedTags)

	builder := &RouteBuilder{
		method:   strings.ToUpper(method),
		path:     fullPath,
		router:   router,
		registry: registry,
		metadata: &RouteMetadata{
			Tags:      tags,
			Responses: make(map[int]ResponseMetadata),
			Examples:  make(map[int]interface{}),
		},
	}

	// Ensure base metadata is visible even when no RouteBuilder methods are chained.
	builder.finalize()
	return builder
}

func resolveRegisteredPath(router fiber.Router, path string) string {
	if group, ok := router.(*fiber.Group); ok {
		return joinRoutePath(group.Prefix, path)
	}
	return joinRoutePath("", path)
}

func joinRoutePath(prefix, path string) string {
	prefix = strings.TrimSpace(prefix)
	path = strings.TrimSpace(path)

	if prefix == "" {
		if path == "" {
			return "/"
		}
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 {
		prefix = strings.TrimSuffix(prefix, "/")
	}

	if path == "" || path == "/" {
		return prefix
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return prefix + path
}

// finalize registers the metadata in the global registry
func (b *RouteBuilder) finalize() {
	if b.registry == nil {
		return
	}
	b.registry.RegisterRoute(b.method, b.path, b.metadata)
}

// Name sets the operation ID.
func (b *RouteBuilder) Name(name string) *RouteBuilder {
	b.metadata.OperationID = name
	b.finalize()
	return b
}

// TaggedName sets operationId prefixed by first tag, e.g. Users_GetUserByID.
// If no tag is available, falls back to plain Name behavior.
func (b *RouteBuilder) TaggedName(name string) *RouteBuilder {
	prefix := ""
	if len(b.metadata.Tags) > 0 {
		prefix = sanitizeOperationIDPart(b.metadata.Tags[0])
	}
	name = sanitizeOperationIDPart(name)
	if prefix == "" {
		b.metadata.OperationID = name
	} else {
		b.metadata.OperationID = prefix + "_" + name
	}
	b.finalize()
	return b
}

// Tags sets tags for grouping in Swagger UI.
func (b *RouteBuilder) Tags(tags ...string) *RouteBuilder {
	b.metadata.Tags = append(b.metadata.Tags, tags...)
	b.finalize()
	return b
}

// Summary sets the operation summary.
func (b *RouteBuilder) Summary(summary string) *RouteBuilder {
	b.metadata.Summary = summary
	b.finalize()
	return b
}

// Description sets the operation description.
func (b *RouteBuilder) Description(description string) *RouteBuilder {
	b.metadata.Description = description
	b.finalize()
	return b
}

// With applies reusable route options in order.
func (b *RouteBuilder) With(opts ...RouteOption) *RouteBuilder {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		b = opt(b)
	}
	return b
}

// Body sets the request body type using type inference.
//
// Usage: .Body(CreateUserRequest{})
func (b *RouteBuilder) Body(requestType any) *RouteBuilder {
	b.setRequestBody(requestType, true, false, ContentTypeApplicationJSON)
	b.finalize()
	return b
}

// BodyAs sets the request body type and content types.
//
// Usage: .BodyAs(CreateUserRequest{}, "application/json", "application/xml")
func (b *RouteBuilder) BodyAs(requestType any, contentTypes ...string) *RouteBuilder {
	b.setRequestBody(requestType, true, false, contentTypes...)
	b.finalize()
	return b
}

// BodyOptional sets an optional request body type.
//
// Usage: .BodyOptional(SearchRequest{})
func (b *RouteBuilder) BodyOptional(requestType any) *RouteBuilder {
	b.setRequestBody(requestType, false, false, ContentTypeApplicationJSON)
	b.finalize()
	return b
}

// BodyAsOptional sets an optional request body type and content types.
//
// Usage: .BodyAsOptional(SearchRequest{}, "application/json")
func (b *RouteBuilder) BodyAsOptional(requestType any, contentTypes ...string) *RouteBuilder {
	b.setRequestBody(requestType, false, false, contentTypes...)
	b.finalize()
	return b
}

// Form sets a required x-www-form-urlencoded request body type.
func (b *RouteBuilder) Form(requestType any) *RouteBuilder {
	b.setRequestBody(requestType, true, false, ContentTypeApplicationXWWWFormURLEncoded)
	b.finalize()
	return b
}

// FormOptional sets an optional x-www-form-urlencoded request body type.
func (b *RouteBuilder) FormOptional(requestType any) *RouteBuilder {
	b.setRequestBody(requestType, false, false, ContentTypeApplicationXWWWFormURLEncoded)
	b.finalize()
	return b
}

// Multipart sets a required multipart/form-data request body type.
func (b *RouteBuilder) Multipart(requestType any) *RouteBuilder {
	b.setRequestBody(requestType, true, false, ContentTypeMultipartFormData)
	b.finalize()
	return b
}

// MultipartOptional sets an optional multipart/form-data request body type.
func (b *RouteBuilder) MultipartOptional(requestType any) *RouteBuilder {
	b.setRequestBody(requestType, false, false, ContentTypeMultipartFormData)
	b.finalize()
	return b
}

// BodyAtLeastOne sets a required request body and enforces that at least one field is present.
// Useful for PATCH-style DTOs with optional fields.
func (b *RouteBuilder) BodyAtLeastOne(requestType any) *RouteBuilder {
	b.setRequestBody(requestType, true, true, ContentTypeApplicationJSON)
	b.finalize()
	return b
}

// BodyAsAtLeastOne sets a required request body with content types and at-least-one-field constraint.
func (b *RouteBuilder) BodyAsAtLeastOne(requestType any, contentTypes ...string) *RouteBuilder {
	b.setRequestBody(requestType, true, true, contentTypes...)
	b.finalize()
	return b
}

// AtLeastOneBodyField enables at-least-one-field constraint for an already configured body.
func (b *RouteBuilder) AtLeastOneBodyField() *RouteBuilder {
	if b.metadata.RequestBody == nil {
		b.metadata.RequestBody = &RequestBodyMetadata{
			Required:          true,
			ContentTypes:      []string{ContentTypeApplicationJSON},
			Content:           map[string]reflect.Type{ContentTypeApplicationJSON: nil},
			RequireAtLeastOne: true,
		}
	} else {
		b.metadata.RequestBody.RequireAtLeastOne = true
	}
	b.finalize()
	return b
}

// Query sets query parameter type using struct tag inference.
//
// Usage: .Query(ListUsersQuery{})
func (b *RouteBuilder) Query(queryType any) *RouteBuilder {
	b.metadata.QueryType = reflect.TypeOf(queryType)
	b.finalize()
	return b
}

// Header adds an optional request header parameter.
// Usage: .Header("X-Tenant-ID", "", "Tenant identifier")
func (b *RouteBuilder) Header(name string, paramType any, description ...string) *RouteBuilder {
	b.addParameterMetadata("header", name, paramType, false, firstDescription(description...), "")
	b.finalize()
	return b
}

// HeaderRequired adds a required request header parameter.
// Usage: .HeaderRequired("X-Tenant-ID", "", "Tenant identifier")
func (b *RouteBuilder) HeaderRequired(name string, paramType any, description ...string) *RouteBuilder {
	b.addParameterMetadata("header", name, paramType, true, firstDescription(description...), "")
	b.finalize()
	return b
}

// HeaderExt adds an optional request header parameter with OpenAPI extensions.
// Usage: .HeaderExt("X-Tenant-ID", "", "x-nullable,x-owner=platform", "Tenant identifier")
func (b *RouteBuilder) HeaderExt(name string, paramType any, extensions string, description ...string) *RouteBuilder {
	b.addParameterMetadata("header", name, paramType, false, firstDescription(description...), extensions)
	b.finalize()
	return b
}

// HeaderRequiredExt adds a required request header parameter with OpenAPI extensions.
func (b *RouteBuilder) HeaderRequiredExt(name string, paramType any, extensions string, description ...string) *RouteBuilder {
	b.addParameterMetadata("header", name, paramType, true, firstDescription(description...), extensions)
	b.finalize()
	return b
}

// Headers adds optional request headers from name->type mappings.
func (b *RouteBuilder) Headers(values map[string]any) *RouteBuilder {
	for name, valueType := range values {
		b.addParameterMetadata("header", name, valueType, false, "", "")
	}
	b.finalize()
	return b
}

// HeadersRequired adds required request headers from name->type mappings.
func (b *RouteBuilder) HeadersRequired(values map[string]any) *RouteBuilder {
	for name, valueType := range values {
		b.addParameterMetadata("header", name, valueType, true, "", "")
	}
	b.finalize()
	return b
}

// Cookie adds an optional request cookie parameter.
// Usage: .Cookie("session_id", "", "Session cookie")
func (b *RouteBuilder) Cookie(name string, paramType any, description ...string) *RouteBuilder {
	b.addParameterMetadata("cookie", name, paramType, false, firstDescription(description...), "")
	b.finalize()
	return b
}

// CookieRequired adds a required request cookie parameter.
// Usage: .CookieRequired("session_id", "", "Session cookie")
func (b *RouteBuilder) CookieRequired(name string, paramType any, description ...string) *RouteBuilder {
	b.addParameterMetadata("cookie", name, paramType, true, firstDescription(description...), "")
	b.finalize()
	return b
}

// CookieExt adds an optional request cookie parameter with OpenAPI extensions.
func (b *RouteBuilder) CookieExt(name string, paramType any, extensions string, description ...string) *RouteBuilder {
	b.addParameterMetadata("cookie", name, paramType, false, firstDescription(description...), extensions)
	b.finalize()
	return b
}

// CookieRequiredExt adds a required request cookie parameter with OpenAPI extensions.
func (b *RouteBuilder) CookieRequiredExt(name string, paramType any, extensions string, description ...string) *RouteBuilder {
	b.addParameterMetadata("cookie", name, paramType, true, firstDescription(description...), extensions)
	b.finalize()
	return b
}

// Cookies adds optional request cookies from name->type mappings.
func (b *RouteBuilder) Cookies(values map[string]any) *RouteBuilder {
	for name, valueType := range values {
		b.addParameterMetadata("cookie", name, valueType, false, "", "")
	}
	b.finalize()
	return b
}

// CookiesRequired adds required request cookies from name->type mappings.
func (b *RouteBuilder) CookiesRequired(values map[string]any) *RouteBuilder {
	for name, valueType := range values {
		b.addParameterMetadata("cookie", name, valueType, true, "", "")
	}
	b.finalize()
	return b
}

// SetHeaders adds response header documentation for a status code.
// Usage: .SetHeaders(201, "Location", "")
func (b *RouteBuilder) SetHeaders(statusCode int, name string, valueType any, description ...string) *RouteBuilder {
	if strings.TrimSpace(name) == "" {
		return b
	}
	b.ensureResponseMaps()
	resp := b.metadata.Responses[statusCode]
	if resp.Headers == nil {
		resp.Headers = make(map[string]ResponseHeaderMetadata)
	}
	resp.Headers[name] = ResponseHeaderMetadata{
		Type:        reflect.TypeOf(valueType),
		Description: firstDescription(description...),
	}
	b.metadata.Responses[statusCode] = resp
	b.finalize()
	return b
}

// SetCookies adds response Set-Cookie header documentation for a status code.
// Usage: .SetCookies(200, "session_id")
func (b *RouteBuilder) SetCookies(statusCode int, cookieName string, description ...string) *RouteBuilder {
	desc := firstDescription(description...)
	cookieName = strings.TrimSpace(cookieName)
	if cookieName != "" {
		if desc == "" {
			desc = "Set-Cookie for " + cookieName
		} else {
			desc = desc + " (cookie: " + cookieName + ")"
		}
	}
	return b.SetHeaders(statusCode, "Set-Cookie", "", desc)
}

// Security adds a security requirement with no scopes.
// Usage: .Security("BearerAuth")
func (b *RouteBuilder) Security(scheme string) *RouteBuilder {
	return b.Scopes(scheme)
}

// Scopes adds a security requirement with scopes.
// Usage: .Scopes("OAuth2", "read:users")
func (b *RouteBuilder) Scopes(scheme string, scopes ...string) *RouteBuilder {
	scheme = strings.TrimSpace(scheme)
	if scheme == "" {
		return b
	}

	copiedScopes := append([]string(nil), scopes...)
	key := securityRequirementKey(SecurityRequirement{
		Scheme: scheme,
		Scopes: copiedScopes,
	})
	for _, existing := range b.metadata.Security {
		if securityRequirementKey(existing) == key {
			return b
		}
	}
	b.metadata.Security = append(b.metadata.Security, SecurityRequirement{
		Scheme: scheme,
		Scopes: copiedScopes,
	})
	b.finalize()
	return b
}

func securityRequirementKey(req SecurityRequirement) string {
	scheme := strings.TrimSpace(req.Scheme)
	scopes := make([]string, 0, len(req.Scopes))
	for _, scope := range req.Scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scheme + "|" + strings.Join(scopes, ",")
}

// Policies adds policy names used by authz policy governance.
// Usage: .Policies("orders.read", "orders.write")
func (b *RouteBuilder) Policies(policies ...string) *RouteBuilder {
	if len(policies) == 0 {
		return b
	}
	seen := make(map[string]struct{}, len(b.metadata.Policies)+len(policies))
	for _, p := range b.metadata.Policies {
		p = strings.TrimSpace(p)
		if p != "" {
			seen[p] = struct{}{}
		}
	}
	for _, p := range policies {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		b.metadata.Policies = append(b.metadata.Policies, p)
	}
	b.finalize()
	return b
}

// Public marks the route as explicitly public (no security requirements).
func (b *RouteBuilder) Public() *RouteBuilder {
	b.metadata.Security = []SecurityRequirement{}
	b.finalize()
	return b
}

// Produces sets a response type for a status code using type inference
// Usage: .Produces(User{}, 200)
func (b *RouteBuilder) Produces(responseType any, statusCode int) *RouteBuilder {
	b.setResponse(statusCode, responseType, inferResponseContentType(responseType), "", false)
	b.finalize()
	return b
}

// ProducesWithDescription sets a response type and custom description for a status code.
func (b *RouteBuilder) ProducesWithDescription(responseType any, statusCode int, description string) *RouteBuilder {
	b.setResponse(statusCode, responseType, inferResponseContentType(responseType), description, true)
	b.finalize()
	return b
}

// ProducesAs sets a response type and content type for a status code.
// Usage: .ProducesAs(string(""), 200, "text/plain")
func (b *RouteBuilder) ProducesAs(responseType any, statusCode int, contentType string) *RouteBuilder {
	if contentType == "" {
		contentType = inferResponseContentType(responseType)
	}
	b.setResponse(statusCode, responseType, contentType, "", false)
	b.finalize()
	return b
}

// NoContent marks a status response as having no response body.
// Usage: .NoContent(204)
func (b *RouteBuilder) NoContent(statusCode int) *RouteBuilder {
	b.ensureResponseMaps()
	resp := b.metadata.Responses[statusCode]
	resp.NoContent = true
	b.metadata.Responses[statusCode] = resp
	b.finalize()
	return b
}

// ProducesWithExample sets a response type with a custom example
// Usage: .ProducesWithExample(User{ID: "123"}, 200)
func (b *RouteBuilder) ProducesWithExample(example any, statusCode int) *RouteBuilder {
	b.setResponse(statusCode, example, inferResponseContentType(example), "", false)
	b.metadata.Examples[statusCode] = example
	b.finalize()
	return b
}

// Ok sets a 200 response schema.
func (b *RouteBuilder) Ok(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 200, firstDescription(description...))
}

// Create sets a 201 response schema.
func (b *RouteBuilder) Create(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 201, firstDescription(description...))
}

// Accepted sets a 202 response schema.
func (b *RouteBuilder) Accepted(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 202, firstDescription(description...))
}

// BadRequest sets a 400 response schema.
func (b *RouteBuilder) BadRequest(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 400, firstDescription(description...))
}

// Unauthorized sets a 401 response schema.
func (b *RouteBuilder) Unauthorized(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 401, firstDescription(description...))
}

// Forbidden sets a 403 response schema.
func (b *RouteBuilder) Forbidden(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 403, firstDescription(description...))
}

// NotFound sets a 404 response schema.
func (b *RouteBuilder) NotFound(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 404, firstDescription(description...))
}

// Conflict sets a 409 response schema.
func (b *RouteBuilder) Conflict(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 409, firstDescription(description...))
}

// UnprocessableEntity sets a 422 response schema.
func (b *RouteBuilder) UnprocessableEntity(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 422, firstDescription(description...))
}

// InternalError sets a 500 response schema.
func (b *RouteBuilder) InternalError(responseType any, description ...string) *RouteBuilder {
	return b.ProducesWithDescription(responseType, 500, firstDescription(description...))
}

// StandardErrors adds common error responses (400/401/403/404/500) using Error schema.
func (b *RouteBuilder) StandardErrors() *RouteBuilder {
	b.ensureResponseMaps()
	b.addErrorResponseIfMissing(400, "Bad request")
	b.addErrorResponseIfMissing(401, "Unauthorized")
	b.addErrorResponseIfMissing(403, "Forbidden")
	b.addErrorResponseIfMissing(404, "Resource not found")
	b.addErrorResponseIfMissing(500, "Internal server error")
	b.finalize()
	return b
}

// ValidationErrors adds validation-focused error responses (400/422) using Error schema.
func (b *RouteBuilder) ValidationErrors() *RouteBuilder {
	b.ensureResponseMaps()
	b.addErrorResponseIfMissing(400, "Invalid request")
	b.addErrorResponseIfMissing(422, "Validation failed")
	b.finalize()
	return b
}

// Paginated adds standard pagination query parameters and a paginated 200 response envelope.
// The generated response shape is:
// { "items": [...], "total": 123, "next_cursor": "..." }.
func (b *RouteBuilder) Paginated(itemType any) *RouteBuilder {
	b.metadata.Pagination = &PaginationMetadata{
		ItemType: reflect.TypeOf(itemType),
	}
	b.finalize()
	return b
}

func (b *RouteBuilder) setRequestBody(requestType any, required bool, requireAtLeastOne bool, contentTypes ...string) {
	if len(contentTypes) == 0 {
		contentTypes = []string{ContentTypeApplicationJSON}
	}
	typ := reflect.TypeOf(requestType)

	if b.metadata.RequestBody == nil {
		b.metadata.RequestBody = &RequestBodyMetadata{
			Type:              typ,
			Required:          required,
			RequireAtLeastOne: requireAtLeastOne,
			Content:           make(map[string]reflect.Type, len(contentTypes)),
		}
	} else {
		b.metadata.RequestBody.Type = typ
		b.metadata.RequestBody.Required = required
		b.metadata.RequestBody.RequireAtLeastOne = requireAtLeastOne
		if b.metadata.RequestBody.Content == nil {
			b.metadata.RequestBody.Content = make(map[string]reflect.Type, len(contentTypes))
		}
	}

	existingTypes := make(map[string]struct{}, len(b.metadata.RequestBody.ContentTypes)+len(contentTypes))
	for _, ct := range b.metadata.RequestBody.ContentTypes {
		ct = strings.TrimSpace(ct)
		if ct == "" {
			continue
		}
		existingTypes[ct] = struct{}{}
	}
	for _, ct := range contentTypes {
		ct = strings.TrimSpace(ct)
		if ct == "" {
			ct = ContentTypeApplicationJSON
		}
		if _, exists := existingTypes[ct]; !exists {
			b.metadata.RequestBody.ContentTypes = append(b.metadata.RequestBody.ContentTypes, ct)
			existingTypes[ct] = struct{}{}
		}
		b.metadata.RequestBody.Content[ct] = typ
	}
}

func firstDescription(description ...string) string {
	if len(description) == 0 {
		return ""
	}
	return description[0]
}

func sanitizeOperationIDPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return nonWordOperationIDChars.ReplaceAllString(v, "_")
}

func (b *RouteBuilder) ensureResponseMaps() {
	if b.metadata.Responses == nil {
		b.metadata.Responses = make(map[int]ResponseMetadata)
	}
	if b.metadata.Examples == nil {
		b.metadata.Examples = make(map[int]interface{})
	}
}

func (b *RouteBuilder) addErrorResponseIfMissing(statusCode int, message string) {
	if _, exists := b.metadata.Responses[statusCode]; exists {
		return
	}

	b.metadata.Responses[statusCode] = ResponseMetadata{
		Type:        reflect.TypeOf(Error{}),
		ContentType: ContentTypeApplicationJSON,
		Content: map[string]reflect.Type{
			ContentTypeApplicationJSON: reflect.TypeOf(Error{}),
		},
	}
	b.metadata.Examples[statusCode] = Error{
		Error: ErrorDetail{
			Code:    strings.ToUpper(strings.ReplaceAll(message, " ", "_")),
			Message: message,
		},
	}
}

func (b *RouteBuilder) setResponse(statusCode int, responseType any, contentType, description string, hasDescription bool) {
	if contentType == "" {
		contentType = inferResponseContentType(responseType)
	}

	b.ensureResponseMaps()
	resp := b.metadata.Responses[statusCode]
	respType := reflect.TypeOf(responseType)

	// Preserve legacy fields for compatibility and for callers that inspect a single value.
	resp.Type = respType
	resp.ContentType = contentType
	resp.NoContent = statusCode == 204
	if hasDescription {
		resp.Description = description
	}

	if resp.Content == nil {
		resp.Content = make(map[string]reflect.Type)
		if resp.ContentType != "" && resp.Type != nil {
			resp.Content[resp.ContentType] = resp.Type
		}
	}
	if resp.ContentVariants == nil {
		resp.ContentVariants = make(map[string][]reflect.Type)
	}
	if contentType != "" {
		if existing, exists := resp.Content[contentType]; exists {
			resp.ContentVariants[contentType] = appendUniqueResponseType(resp.ContentVariants[contentType], existing)
		}
		resp.ContentVariants[contentType] = appendUniqueResponseType(resp.ContentVariants[contentType], respType)
		resp.Content[contentType] = respType
	}

	b.metadata.Responses[statusCode] = resp
}

func appendUniqueResponseType(existing []reflect.Type, t reflect.Type) []reflect.Type {
	if t == nil {
		return existing
	}
	for _, item := range existing {
		if item == t {
			return existing
		}
	}
	return append(existing, t)
}

func inferResponseContentType(responseType any) string {
	t := reflect.TypeOf(responseType)
	if t == nil {
		return ContentTypeApplicationJSON
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return ContentTypeTextPlain
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return ContentTypeApplicationOctetStream
		}
	}
	return ContentTypeApplicationJSON
}

func (b *RouteBuilder) addParameterMetadata(in, name string, paramType any, required bool, description, extensions string) {
	if strings.TrimSpace(name) == "" {
		return
	}

	b.metadata.Parameters = append(b.metadata.Parameters, ParameterMetadata{
		Name:        name,
		In:          in,
		Type:        reflect.TypeOf(paramType),
		Required:    required,
		Description: description,
		Extensions:  extensions,
	})
}
