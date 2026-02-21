package autoswag

import (
	"fmt"
	"mime/multipart"
	stdpath "path"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bronystylecrazy/ultrastructure/web"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OpenAPISpec represents a runtime-generated OpenAPI specification
type OpenAPISpec struct {
	OpenAPI    string                            `json:"openapi"`
	Info       OpenAPIInfo                       `json:"info"`
	Paths      map[string]map[string]interface{} `json:"paths"`
	Security   []map[string][]string             `json:"security,omitempty"`
	Tags       []OpenAPITag                      `json:"tags,omitempty"`
	Components *OpenAPIComponents                `json:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Version        string          `json:"version"`
	TermsOfService string          `json:"termsOfService,omitempty"`
	Contact        *OpenAPIContact `json:"contact,omitempty"`
	License        *OpenAPILicense `json:"license,omitempty"`
}

type OpenAPIContact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type OpenAPILicense struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type OpenAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type OpenAPIComponents struct {
	Schemas         map[string]interface{} `json:"schemas,omitempty"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes,omitempty"`
}

type OpenAPIBuildOptions struct {
	SecuritySchemes     map[string]interface{}
	DefaultSecurity     []SecurityRequirement
	TagDescriptions     map[string]string
	PackageTagTransform func(string) string
	IncludeDiagnostics  bool
	DiagnosticsSeverity string
	FailOnDiagnostics   bool
	TermsOfService      string
	Contact             *OpenAPIContact
	License             *OpenAPILicense
	PreHook             HookFunc
	Hook                HookFunc
	PostHook            HookFunc
	ExtraModels         []reflect.Type
}

// RouteInfo contains metadata about a registered route
type RouteInfo struct {
	Method             string
	Path               string
	Handler            string
	HandlerType        reflect.Type
	HandlerName        string
	HandlerPackagePath string
}

// InspectFiberRoutes extracts route information from a Fiber app
func InspectFiberRoutes(app *fiber.App, logger *zap.Logger) []RouteInfo {
	routes := []RouteInfo{}

	// Get all registered routes from Fiber.
	// Passing true filters out middleware routes registered via app.Use(...),
	// which would otherwise appear as all HTTP methods on "/".
	for _, route := range app.GetRoutes(true) {
		// Skip internal routes
		if strings.HasPrefix(route.Path, "/_") {
			continue
		}

		routeInfo := RouteInfo{
			Method: route.Method,
			Path:   route.Path,
		}

		// Try to extract handler information
		if len(route.Handlers) > 0 {
			handler := route.Handlers[len(route.Handlers)-1]
			handlerType := reflect.TypeOf(handler)
			routeInfo.Handler = handlerType.String()
			routeInfo.HandlerType = handlerType
			routeInfo.HandlerName = resolveHandlerName(handler)
			routeInfo.HandlerPackagePath = resolveHandlerPackagePath(routeInfo.HandlerName)
		}

		routes = append(routes, routeInfo)

		if logger != nil {
			logger.Debug("discovered route",
				zap.String("method", routeInfo.Method),
				zap.String("path", routeInfo.Path),
				zap.String("handler", routeInfo.Handler),
				zap.String("handler_name", routeInfo.HandlerName),
				zap.String("handler_package", routeInfo.HandlerPackagePath),
			)
		}
	}

	return routes
}

// BuildOpenAPISpec generates an OpenAPI 3.0 spec from route information
func BuildOpenAPISpec(routes []RouteInfo, config Config) *OpenAPISpec {
	return BuildOpenAPISpecWithOptions(routes, config, OpenAPIBuildOptions{})
}

// BuildOpenAPISpecWithSecuritySchemes generates an OpenAPI spec with optional global security schemes.
func BuildOpenAPISpecWithSecuritySchemes(routes []RouteInfo, config Config, securitySchemes map[string]interface{}) *OpenAPISpec {
	return BuildOpenAPISpecWithOptions(routes, config, OpenAPIBuildOptions{
		SecuritySchemes: securitySchemes,
	})
}

// BuildOpenAPISpecWithSecurity generates an OpenAPI spec with optional global security schemes and default security requirements.
func BuildOpenAPISpecWithSecurity(routes []RouteInfo, config Config, securitySchemes map[string]interface{}, defaultSecurity []SecurityRequirement) *OpenAPISpec {
	return BuildOpenAPISpecWithOptions(routes, config, OpenAPIBuildOptions{
		SecuritySchemes: securitySchemes,
		DefaultSecurity: defaultSecurity,
	})
}

// BuildOpenAPISpecWithRegistry generates an OpenAPI spec using a specific metadata registry.
func BuildOpenAPISpecWithRegistry(routes []RouteInfo, config Config, registry *MetadataRegistry) *OpenAPISpec {
	return buildOpenAPISpecWithRegistryAndOptions(routes, config, registry, OpenAPIBuildOptions{})
}

// BuildOpenAPISpecWithOptions generates an OpenAPI spec with extended metadata options.
func BuildOpenAPISpecWithOptions(routes []RouteInfo, config Config, opts OpenAPIBuildOptions) *OpenAPISpec {
	return buildOpenAPISpecWithRegistryAndOptions(routes, config, GetGlobalRegistry(), opts)
}

// BuildOpenAPISpecWithRegistryAndOptions generates an OpenAPI spec
// using a specific metadata registry and extended metadata options.
func BuildOpenAPISpecWithRegistryAndOptions(routes []RouteInfo, config Config, registry *MetadataRegistry, opts OpenAPIBuildOptions) *OpenAPISpec {
	return buildOpenAPISpecWithRegistryAndOptions(routes, config, registry, opts)
}

func buildOpenAPISpecWithRegistryAndOptions(routes []RouteInfo, config Config, registry *MetadataRegistry, opts OpenAPIBuildOptions) *OpenAPISpec {
	if registry == nil {
		registry = GetGlobalRegistry()
	}

	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       config.Name,
			Description: fmt.Sprintf("Auto-generated API documentation for %s", config.Name),
			Version:     "1.0.0",
		},
		Paths: make(map[string]map[string]interface{}),
		Components: &OpenAPIComponents{
			Schemas: make(map[string]interface{}),
		},
	}
	spec.Info.TermsOfService = opts.TermsOfService
	spec.Info.Contact = opts.Contact
	spec.Info.License = opts.License

	// Create schema extractor for deep type analysis
	extractor := NewSchemaExtractor()
	hookModels := NewSwaggerModelRegistry()
	if len(opts.ExtraModels) > 0 {
		initial := make([]any, 0, len(opts.ExtraModels))
		for _, t := range opts.ExtraModels {
			initial = append(initial, t)
		}
		hookModels.Add(initial...)
	}
	usedOperationIDs := make(map[string]struct{})
	accumTagDescriptions := map[string]string{}
	for k, v := range opts.TagDescriptions {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		accumTagDescriptions[k] = v
	}

	for _, route := range routes {
		openAPIPath := normalizeOpenAPIPath(route.Path)
		if _, exists := spec.Paths[openAPIPath]; !exists {
			spec.Paths[openAPIPath] = make(map[string]interface{})
		}

		method := strings.ToUpper(route.Method)
		methodKey := strings.ToLower(route.Method)
		// Fiber can expose duplicate route entries for the same path+method.
		// OpenAPI supports a single operation per path+method, so keep the first one.
		if _, exists := spec.Paths[openAPIPath][methodKey]; exists {
			continue
		}
		hasRequestBody := method == "POST" || method == "PUT" || method == "PATCH"
		operation := map[string]interface{}{}

		// Try to get metadata from registry
		metadata := cloneRouteMetadata(registry.GetRoute(method, route.Path))
		runHook := opts.Hook
		if runHook == nil {
			runHook = getRegisteredHook()
		}

		ctx := &SwaggerContext{
			Route:            route,
			Method:           method,
			Path:             openAPIPath,
			Metadata:         metadata,
			Models:           hookModels,
			Spec:             spec,
			Operation:        operation,
			conflictSeverity: opts.DiagnosticsSeverity,
		}

		if opts.PreHook != nil {
			opts.PreHook(ctx)
			metadata = ctx.Metadata
		}
		if runHook != nil {
			if metadata == nil {
				metadata = &RouteMetadata{}
				ctx.Metadata = metadata
			}
			runHook(ctx)
			metadata = ctx.Metadata
		}
		operationTags := extractOperationTags(route, metadata, opts.PackageTagTransform)
		if _, exists := operation["summary"]; !exists {
			operation["summary"] = generateSummaryFromMetadata(route, metadata)
		}

		// Use metadata if available
		var baseOperationID string
		generatedOperationID := true
		if metadata != nil {
			for tag, description := range metadata.TagDescriptions {
				tag = strings.TrimSpace(tag)
				description = strings.TrimSpace(description)
				if tag == "" || description == "" {
					continue
				}
				accumTagDescriptions[tag] = description
			}
			if metadata.OperationID != "" {
				baseOperationID = metadata.OperationID
				generatedOperationID = false
			}
			if metadata.Description != "" {
				if _, exists := operation["description"]; exists {
					// Keep explicit operation override from customize hooks.
				} else {
					operation["description"] = metadata.Description
				}
			}
			if len(operationTags) > 0 {
				if _, exists := operation["tags"]; !exists {
					operation["tags"] = operationTags
				}
			}
			if metadata.Security != nil {
				if _, exists := operation["security"]; !exists {
					security := buildOperationSecurity(metadata.Security)
					if len(security) == 0 {
						// Explicitly empty operation security means "public route"
						// and overrides any global/default requirements.
						operation["security"] = []map[string][]string{}
					} else {
						operation["security"] = security
					}
				}
			}
			if _, exists := operation["x-scopes"]; !exists {
				if scopes := collectRouteScopes(metadata); len(scopes) > 0 {
					operation["x-scopes"] = scopes
				}
			}
			if _, exists := operation["x-policies"]; !exists {
				if policies := normalizeOperationStrings(metadata.Policies); len(policies) > 0 {
					operation["x-policies"] = policies
				}
			}
		}
		if existingOperationID, ok := operation["operationId"].(string); ok && strings.TrimSpace(existingOperationID) != "" {
			baseOperationID = existingOperationID
			generatedOperationID = false
		}
		if strings.TrimSpace(baseOperationID) == "" {
			baseOperationID = generateDeterministicOperationID(method, route.Path)
			generatedOperationID = true
		}
		baseOperationID = applyGlobalOperationIDTagPrefix(baseOperationID, operationTags)
		baseOperationID = applyRegisteredOperationIDHook(OperationIDHookContext{
			Route:       route,
			Metadata:    metadata,
			Tags:        append([]string(nil), operationTags...),
			OperationID: baseOperationID,
			Generated:   generatedOperationID,
		})
		operation["operationId"] = makeUniqueOperationID(baseOperationID, usedOperationIDs)

		// Generate responses using metadata if available.
		if _, exists := operation["responses"]; !exists {
			if metadata != nil && len(metadata.Responses) > 0 {
				operation["responses"] = generateResponsesFromMetadata(metadata, extractor)
			} else {
				operation["responses"] = generateResponses(route, extractor)
			}
		}
		if metadata != nil && metadata.Pagination != nil {
			_, hasExplicit200 := metadata.Responses[200]
			applyPaginationResponse(operation, metadata.Pagination, extractor, hasExplicit200)
		}

		// Add request body for methods that typically support bodies, or whenever metadata declares one.
		if _, exists := operation["requestBody"]; !exists {
			if requestBody := buildRequestBody(metadata, hasRequestBody, extractor); requestBody != nil {
				operation["requestBody"] = requestBody
			}
		}

		// Add path/query/custom parameters.
		if _, exists := operation["parameters"]; !exists {
			if params := extractParameters(route.Path, metadata); len(params) > 0 {
				operation["parameters"] = params
			}
		}

		// Add tags (from metadata or path fallback).
		if len(operationTags) > 0 {
			if _, exists := operation["tags"]; !exists {
				operation["tags"] = operationTags
			}
		}
		if opts.PostHook != nil {
			opts.PostHook(ctx)
			metadata = ctx.Metadata
		}
		if opts.IncludeDiagnostics && len(ctx.Warnings) > 0 {
			operation["x-autoswag-warnings"] = append([]string(nil), ctx.Warnings...)
			operation["x-autoswag-diagnostics"] = diagnosticsToOpenAPI(ctx.Diagnostics)
		}
		if opts.FailOnDiagnostics && hasErrorDiagnostics(ctx.Diagnostics) {
			panic(fmt.Sprintf("autoswag conflict diagnostics error on %s %s", method, openAPIPath))
		}

		spec.Paths[openAPIPath][methodKey] = operation
	}

	// Add explicitly registered models (including hook-added models) and extracted schemas to components.
	addExtraModelSchemas(extractor, hookModels.Types())
	for name, schema := range extractor.GetSchemas() {
		spec.Components.Schemas[name] = schema
	}

	// Add global security schemes if configured.
	if len(opts.SecuritySchemes) > 0 {
		spec.Components.SecuritySchemes = make(map[string]interface{}, len(opts.SecuritySchemes))
		for name, scheme := range opts.SecuritySchemes {
			spec.Components.SecuritySchemes[name] = scheme
		}
	}

	if security := buildOperationSecurity(opts.DefaultSecurity); len(security) > 0 {
		spec.Security = security
	}

	spec.Tags = buildSpecTags(spec.Paths, accumTagDescriptions)

	return spec
}

func addExtraModelSchemas(extractor *SchemaExtractor, modelTypes []reflect.Type) {
	if extractor == nil || len(modelTypes) == 0 {
		return
	}

	for _, t := range modelTypes {
		normalized := normalizeSwaggerModelType(t)
		if normalized == nil || isSkippedType(normalized) {
			continue
		}

		name := extractor.getTypeName(normalized)
		if name == "" {
			extractor.ExtractSchemaRef(normalized)
			continue
		}

		// Explicit extra models take precedence over already-generated schemas
		// with the same component name (e.g. AddNamed("ErrorResponse", ...)).
		extractor.schemas[name] = extractor.extractTypeSchema(normalized)
	}
}

func cloneRouteMetadata(meta *RouteMetadata) *RouteMetadata {
	if meta == nil {
		return nil
	}

	cloned := *meta
	if meta.Tags != nil {
		cloned.Tags = append([]string{}, meta.Tags...)
	}
	if meta.TagDescriptions != nil {
		cloned.TagDescriptions = make(map[string]string, len(meta.TagDescriptions))
		for k, v := range meta.TagDescriptions {
			cloned.TagDescriptions[k] = v
		}
	}
	if meta.Parameters != nil {
		cloned.Parameters = append([]ParameterMetadata{}, meta.Parameters...)
	}
	if meta.Security != nil {
		cloned.Security = append([]SecurityRequirement{}, meta.Security...)
	}
	if meta.Policies != nil {
		cloned.Policies = append([]string{}, meta.Policies...)
	}

	if meta.RequestBody != nil {
		req := *meta.RequestBody
		req.ContentTypes = append([]string(nil), meta.RequestBody.ContentTypes...)
		if meta.RequestBody.Content != nil {
			req.Content = make(map[string]reflect.Type, len(meta.RequestBody.Content))
			for contentType, t := range meta.RequestBody.Content {
				req.Content[contentType] = t
			}
		}
		cloned.RequestBody = &req
	}

	if meta.Responses != nil {
		cloned.Responses = make(map[int]ResponseMetadata, len(meta.Responses))
		for code, resp := range meta.Responses {
			respCopy := resp
			if resp.Headers != nil {
				respCopy.Headers = make(map[string]ResponseHeaderMetadata, len(resp.Headers))
				for name, header := range resp.Headers {
					respCopy.Headers[name] = header
				}
			}
			if resp.Content != nil {
				respCopy.Content = make(map[string]reflect.Type, len(resp.Content))
				for contentType, t := range resp.Content {
					respCopy.Content[contentType] = t
				}
			}
			if resp.ContentVariants != nil {
				respCopy.ContentVariants = make(map[string][]reflect.Type, len(resp.ContentVariants))
				for contentType, types := range resp.ContentVariants {
					respCopy.ContentVariants[contentType] = append([]reflect.Type(nil), types...)
				}
			}
			cloned.Responses[code] = respCopy
		}
	}

	if meta.Examples != nil {
		cloned.Examples = make(map[int]interface{}, len(meta.Examples))
		for code, example := range meta.Examples {
			cloned.Examples[code] = example
		}
	}

	if meta.Pagination != nil {
		p := *meta.Pagination
		cloned.Pagination = &p
	}

	return &cloned
}

func extractOperationTags(route RouteInfo, metadata *RouteMetadata, transform func(string) string) []string {
	if metadata != nil && len(metadata.Tags) > 0 {
		return metadata.Tags
	}
	if handlerTag := deriveTagFromHandlerPackage(route.HandlerPackagePath, transform); handlerTag != "" {
		return []string{handlerTag}
	}
	return extractTags(route.Path)
}

func resolveHandlerName(handler any) string {
	v := reflect.ValueOf(handler)
	if !v.IsValid() || v.Kind() != reflect.Func {
		return ""
	}
	fn := runtime.FuncForPC(v.Pointer())
	if fn == nil {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(fn.Name()), "-fm")
}

func resolveHandlerPackagePath(handlerName string) string {
	handlerName = strings.TrimSpace(handlerName)
	if handlerName == "" {
		return ""
	}
	parts := strings.Split(handlerName, "/")
	last := parts[len(parts)-1]
	dot := strings.Index(last, ".")
	if dot <= 0 {
		return ""
	}
	parts[len(parts)-1] = last[:dot]
	return strings.Join(parts, "/")
}

func deriveTagFromHandlerPackage(packagePath string, transform func(string) string) string {
	packagePath = strings.TrimSpace(packagePath)
	if packagePath == "" {
		return ""
	}
	tag := strings.TrimSpace(stdpath.Base(packagePath))
	if tag == "." || tag == "/" {
		return ""
	}
	if transform != nil {
		tag = strings.TrimSpace(transform(tag))
	}
	if tag == "" {
		return ""
	}
	return tag
}

// generateSummary creates a human-readable summary from route info
func generateSummary(route RouteInfo) string {
	method := strings.ToUpper(route.Method)
	path := route.Path

	// Extract resource name from path
	segments := strings.Split(strings.Trim(path, "/"), "/")
	var resource string

	for i := len(segments) - 1; i >= 0; i-- {
		if !strings.HasPrefix(segments[i], ":") && segments[i] != "" {
			resource = segments[i]
			break
		}
	}

	if resource == "" {
		return fmt.Sprintf("%s %s", method, path)
	}

	hasParam := strings.Contains(path, ":")

	switch method {
	case "GET":
		if hasParam {
			return fmt.Sprintf("Get single %s", singularize(resource))
		}
		return fmt.Sprintf("List all %s", resource)
	case "POST":
		return fmt.Sprintf("Create new %s", singularize(resource))
	case "PUT", "PATCH":
		return fmt.Sprintf("Update %s", singularize(resource))
	case "DELETE":
		return fmt.Sprintf("Delete %s", singularize(resource))
	default:
		return fmt.Sprintf("%s %s", method, path)
	}
}

// singularize attempts basic singularization
func singularize(word string) string {
	if strings.HasSuffix(word, "s") && len(word) > 1 {
		return word[:len(word)-1]
	}
	return word
}

// generateResponses creates response definitions with proper schemas
func generateResponses(route RouteInfo, extractor *SchemaExtractor) map[string]interface{} {
	method := strings.ToUpper(route.Method)
	defaultErrorRef := ensureDefaultErrorResponseRef(extractor)

	responses := map[string]interface{}{}

	// Success responses
	if method == "DELETE" {
		responses["204"] = map[string]interface{}{
			"description": "Successfully deleted",
		}
		responses["404"] = map[string]interface{}{
			"description": "Resource not found",
		}
	} else if method == "POST" {
		responses["201"] = map[string]interface{}{
			"description": "Successfully created",
			"content": map[string]interface{}{
				web.ContentTypeApplicationJSON: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"example": map[string]interface{}{
							"id":         "123e4567-e89b-12d3-a456-426614174000",
							"created_at": "2024-01-01T00:00:00Z",
						},
					},
				},
			},
		}
	} else {
		responses["200"] = map[string]interface{}{
			"description": "Successful response",
			"content": map[string]interface{}{
				web.ContentTypeApplicationJSON: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"example": map[string]interface{}{
							"data": "Response data",
						},
					},
				},
			},
		}
	}

	// Error responses
	if method == "POST" || method == "PUT" || method == "PATCH" {
		responses["400"] = map[string]interface{}{
			"description": "Bad request - invalid input",
			"content": map[string]interface{}{
				web.ContentTypeApplicationJSON: map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": defaultErrorRef,
					},
					"example": map[string]interface{}{"error": "Invalid request body"},
				},
			},
		}
	}

	if strings.Contains(route.Path, ":") && method != "POST" {
		responses["404"] = map[string]interface{}{
			"description": "Resource not found",
		}
	}

	responses["500"] = map[string]interface{}{
		"description": "Internal server error",
		"content": map[string]interface{}{
			web.ContentTypeApplicationJSON: map[string]interface{}{
				"schema": map[string]interface{}{
					"$ref": defaultErrorRef,
				},
				"example": map[string]interface{}{"error": "Internal server error"},
			},
		},
	}

	return responses
}

// extractPathParams extracts parameter definitions from a path
func extractPathParams(path string) []map[string]interface{} {
	params := []map[string]interface{}{}

	segments := strings.Split(path, "/")
	for _, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			paramName := strings.TrimPrefix(segment, ":")
			params = append(params, map[string]interface{}{
				"name":        paramName,
				"in":          "path",
				"required":    true,
				"description": fmt.Sprintf("The %s identifier", paramName),
				"schema": map[string]interface{}{
					"type":    "string",
					"example": "123e4567-e89b-12d3-a456-426614174000",
				},
			})
		}
	}

	return params
}

// extractParameters extracts path and query parameter definitions.
func extractParameters(path string, metadata *RouteMetadata) []map[string]interface{} {
	params := extractPathParams(path)

	if metadata != nil {
		if metadata.QueryType != nil {
			params = append(params, extractQueryParams(metadata.QueryType)...)
		}
		params = append(params, extractMetadataParams(metadata.Parameters)...)
		if metadata.Pagination != nil {
			params = mergeParameters(params, extractPaginationParams())
		}
	}

	return params
}

// extractQueryParams extracts OpenAPI query parameters from a struct type.
func extractQueryParams(queryType reflect.Type) []map[string]interface{} {
	if queryType == nil {
		return nil
	}

	for queryType.Kind() == reflect.Ptr {
		queryType = queryType.Elem()
	}

	if queryType.Kind() != reflect.Struct {
		return nil
	}

	params := []map[string]interface{}{}
	for i := 0; i < queryType.NumField(); i++ {
		field := queryType.Field(i)
		if !field.IsExported() {
			continue
		}
		if shouldSwaggerIgnoreField(field) {
			continue
		}
		if isSkippedType(field.Type) {
			continue
		}

		name, omitempty, skip := extractQueryFieldName(field)
		if skip {
			continue
		}

		param := map[string]interface{}{
			"name":     name,
			"in":       "query",
			"required": isQueryFieldRequired(field, omitempty),
			"schema":   mapOpenAPIType(field.Type),
		}
		schema := param["schema"].(map[string]interface{})
		if overrideSchema, ok := schemaFromSwaggerTypeTag(field.Tag.Get("swaggertype")); ok {
			schema = overrideSchema
			param["schema"] = schema
		}
		if value, ok := parseTagValue(field.Tag.Get("example"), field.Type); ok {
			schema["example"] = value
		}
		if value, ok := parseTagValue(field.Tag.Get("default"), field.Type); ok {
			schema["default"] = value
		}
		applyExtensionsTag(schema, field.Tag.Get("extensions"))

		if desc := field.Tag.Get("description"); desc != "" {
			param["description"] = desc
		}

		params = append(params, param)
	}

	return params
}

func extractQueryFieldName(field reflect.StructField) (name string, omitempty bool, skip bool) {
	queryTag := field.Tag.Get("query")
	if queryTag == "-" {
		return "", false, true
	}
	if queryTag != "" {
		parts := strings.Split(queryTag, ",")
		if parts[0] != "" {
			name = parts[0]
		}
		for _, p := range parts[1:] {
			if p == "omitempty" {
				omitempty = true
			}
		}
	}

	if name == "" {
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			return "", false, true
		}
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			for _, p := range parts[1:] {
				if p == "omitempty" {
					omitempty = true
				}
			}
		}
	}

	if name == "" {
		name = field.Name
	}

	return name, omitempty, false
}

func isQueryFieldRequired(field reflect.StructField, omitempty bool) bool {
	// Explicit validator tag takes precedence over type/tag heuristics.
	if hasValidateRequired(field.Tag.Get("validate")) {
		return true
	}

	if omitempty {
		return false
	}

	// Pointer query values are treated as optional by default.
	if field.Type.Kind() == reflect.Ptr {
		return false
	}

	return false
}

func hasValidateRequired(validateTag string) bool {
	for _, p := range strings.Split(validateTag, ",") {
		if strings.TrimSpace(p) == "required" {
			return true
		}
	}
	return false
}

func mapOpenAPIType(t reflect.Type) map[string]interface{} {
	originalType := t
	if t == nil {
		return map[string]interface{}{"type": "string"}
	}

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if replaced, ok := resolveReplacedType(t); ok {
		t = replaced
	}

	if t == reflect.TypeOf(time.Time{}) {
		return map[string]interface{}{
			"type":   "string",
			"format": "date-time",
		}
	}

	if t == reflect.TypeOf(uuid.UUID{}) {
		return map[string]interface{}{
			"type":   "string",
			"format": "uuid",
		}
	}

	if t == reflect.TypeOf(multipart.FileHeader{}) {
		return map[string]interface{}{
			"type":   "string",
			"format": "binary",
		}
	}

	switch t.Kind() {
	case reflect.String:
		schema := map[string]interface{}{"type": "string"}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	case reflect.Bool:
		schema := map[string]interface{}{"type": "boolean"}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema := map[string]interface{}{"type": "integer"}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema := map[string]interface{}{"type": "integer", "minimum": 0}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	case reflect.Float32, reflect.Float64:
		schema := map[string]interface{}{"type": "number"}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	default:
		schema := map[string]interface{}{"type": "string"}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	}
}

func extractMetadataParams(metadataParams []ParameterMetadata) []map[string]interface{} {
	if len(metadataParams) == 0 {
		return nil
	}

	params := make([]map[string]interface{}, 0, len(metadataParams))
	for _, p := range metadataParams {
		in := strings.ToLower(strings.TrimSpace(p.In))
		if in != "query" && in != "header" && in != "cookie" {
			continue
		}
		if strings.TrimSpace(p.Name) == "" {
			continue
		}

		param := map[string]interface{}{
			"name":     p.Name,
			"in":       in,
			"required": p.Required,
			"schema":   mapOpenAPIType(p.Type),
		}
		if isSkippedType(p.Type) {
			continue
		}
		schema := param["schema"].(map[string]interface{})
		applyExtensionsTag(schema, p.Extensions)

		if p.Description != "" {
			param["description"] = p.Description
		}

		params = append(params, param)
	}

	return params
}

// extractTags extracts tags from the path
func extractTags(path string) []string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) > 0 && segments[0] != "" && !strings.HasPrefix(segments[0], ":") {
		// Remove "api" prefix if present
		tag := segments[0]
		if tag == "api" && len(segments) > 1 {
			tag = segments[1]
		}
		return []string{tag}
	}
	return nil
}

func normalizeOpenAPIPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") && len(segment) > 1 {
			segments[i] = "{" + strings.TrimPrefix(segment, ":") + "}"
		}
	}
	return strings.Join(segments, "/")
}

func buildRequestBody(metadata *RouteMetadata, hasRequestBody bool, extractor *SchemaExtractor) map[string]interface{} {
	if metadata != nil && metadata.RequestBody != nil {
		content := make(map[string]interface{})
		for contentType, modelType := range resolveRequestBodyContent(metadata.RequestBody) {
			schema := map[string]interface{}{
				"type": "object",
				"example": map[string]interface{}{
					"data": "Request data goes here",
				},
			}
			if modelType != nil {
				schema = extractor.ExtractSchemaRef(modelType)
			}
			if metadata.RequestBody.RequireAtLeastOne {
				schema = applyAtLeastOneFieldConstraintWithRefs(schema, extractor)
			}

			content[contentType] = map[string]interface{}{
				"schema": schema,
			}
		}

		return map[string]interface{}{
			"required":    metadata.RequestBody.Required,
			"description": "Request payload",
			"content":     content,
		}
	}

	if !hasRequestBody {
		return nil
	}

	// Fallback to generic schema for POST/PUT/PATCH without metadata.
	return map[string]interface{}{
		"required":    true,
		"description": "Request payload",
		"content": map[string]interface{}{
			web.ContentTypeApplicationJSON: map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"example": map[string]interface{}{
						"data": "Request data goes here",
					},
				},
			},
		},
	}
}

func resolveRequestBodyContent(requestBody *RequestBodyMetadata) map[string]reflect.Type {
	if requestBody == nil {
		return map[string]reflect.Type{
			web.ContentTypeApplicationJSON: nil,
		}
	}

	if len(requestBody.Content) > 0 {
		out := make(map[string]reflect.Type, len(requestBody.Content))
		for contentType, t := range requestBody.Content {
			normalized := strings.TrimSpace(contentType)
			if normalized == "" {
				normalized = web.ContentTypeApplicationJSON
			}
			out[normalized] = t
		}
		return out
	}

	contentTypes := requestBody.ContentTypes
	if len(contentTypes) == 0 {
		contentTypes = []string{web.ContentTypeApplicationJSON}
	}
	out := make(map[string]reflect.Type, len(contentTypes))
	for _, contentType := range contentTypes {
		normalized := strings.TrimSpace(contentType)
		if normalized == "" {
			normalized = web.ContentTypeApplicationJSON
		}
		out[normalized] = requestBody.Type
	}
	return out
}

func buildOperationSecurity(requirements []SecurityRequirement) []map[string][]string {
	if len(requirements) == 0 {
		return nil
	}

	security := make([]map[string][]string, 0, len(requirements))
	for _, req := range requirements {
		scheme := strings.TrimSpace(req.Scheme)
		if scheme == "" {
			continue
		}

		scopes := append([]string(nil), req.Scopes...)
		security = append(security, map[string][]string{
			scheme: scopes,
		})
	}

	return security
}

func collectRouteScopes(metadata *RouteMetadata) []string {
	if metadata == nil || len(metadata.Security) == 0 {
		return nil
	}
	out := make([]string, 0, len(metadata.Security))
	for _, req := range metadata.Security {
		out = append(out, req.Scopes...)
	}
	return normalizeOperationStrings(out)
}

func normalizeOperationStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func buildSpecTags(paths map[string]map[string]interface{}, tagDescriptions map[string]string) []OpenAPITag {
	tagSet := map[string]struct{}{}
	for _, ops := range paths {
		for _, rawOp := range ops {
			op, ok := rawOp.(map[string]interface{})
			if !ok {
				continue
			}
			rawTags, ok := op["tags"].([]string)
			if !ok {
				continue
			}
			for _, tag := range rawTags {
				if tag == "" {
					continue
				}
				tagSet[tag] = struct{}{}
			}
		}
	}

	for tag := range tagDescriptions {
		if strings.TrimSpace(tag) != "" {
			tagSet[tag] = struct{}{}
		}
	}

	if len(tagSet) == 0 {
		return nil
	}

	names := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		names = append(names, tag)
	}
	sort.Strings(names)

	tags := make([]OpenAPITag, 0, len(names))
	for _, name := range names {
		tags = append(tags, OpenAPITag{
			Name:        name,
			Description: tagDescriptions[name],
		})
	}
	return tags
}

func applyAtLeastOneFieldConstraint(schema map[string]interface{}) {
	if schema == nil {
		return
	}
	if schema["type"] != "object" {
		return
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok || len(properties) == 0 {
		return
	}

	existingRequired := map[string]struct{}{}
	if required, ok := schema["required"].([]string); ok {
		for _, name := range required {
			existingRequired[name] = struct{}{}
		}
	}

	fieldNames := make([]string, 0, len(properties))
	for name := range properties {
		if _, isAlreadyRequired := existingRequired[name]; isAlreadyRequired {
			continue
		}
		fieldNames = append(fieldNames, name)
	}
	if len(fieldNames) == 0 {
		return
	}
	sort.Strings(fieldNames)

	anyOf := make([]map[string]interface{}, 0, len(fieldNames))
	for _, name := range fieldNames {
		anyOf = append(anyOf, map[string]interface{}{
			"required": []string{name},
		})
	}

	schema["anyOf"] = anyOf
}

func applyAtLeastOneFieldConstraintWithRefs(schema map[string]interface{}, extractor *SchemaExtractor) map[string]interface{} {
	if schema == nil {
		return schema
	}

	ref, isRef := schema["$ref"].(string)
	if !isRef {
		applyAtLeastOneFieldConstraint(schema)
		return schema
	}

	fieldNames := extractOptionalFieldNamesFromSchemaRef(ref, extractor)
	if len(fieldNames) == 0 {
		return schema
	}

	anyOf := make([]map[string]interface{}, 0, len(fieldNames))
	for _, name := range fieldNames {
		anyOf = append(anyOf, map[string]interface{}{
			"required": []string{name},
		})
	}

	return map[string]interface{}{
		"allOf": []map[string]interface{}{
			{"$ref": ref},
			{"anyOf": anyOf},
		},
	}
}

func extractOptionalFieldNamesFromSchemaRef(ref string, extractor *SchemaExtractor) []string {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}

	name := strings.TrimPrefix(ref, prefix)
	schemaAny, ok := extractor.GetSchemas()[name]
	if !ok {
		return nil
	}

	schema, ok := schemaAny.(map[string]interface{})
	if !ok {
		return nil
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok || len(properties) == 0 {
		return nil
	}

	requiredSet := map[string]struct{}{}
	if required, ok := schema["required"].([]string); ok {
		for _, field := range required {
			requiredSet[field] = struct{}{}
		}
	}

	fieldNames := make([]string, 0, len(properties))
	for field := range properties {
		if _, required := requiredSet[field]; required {
			continue
		}
		fieldNames = append(fieldNames, field)
	}
	sort.Strings(fieldNames)
	return fieldNames
}

func extractPaginationParams() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "page",
			"in":          "query",
			"required":    false,
			"description": "Page number (1-based)",
			"schema": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
				"default": 1,
				"example": 1,
			},
		},
		{
			"name":        "limit",
			"in":          "query",
			"required":    false,
			"description": "Page size",
			"schema": map[string]interface{}{
				"type":    "integer",
				"minimum": 1,
				"default": 20,
				"example": 20,
			},
		},
		{
			"name":        "sort",
			"in":          "query",
			"required":    false,
			"description": "Sort expression, e.g. -created_at",
			"schema": map[string]interface{}{
				"type":    "string",
				"example": "-created_at",
			},
		},
		{
			"name":        "cursor",
			"in":          "query",
			"required":    false,
			"description": "Opaque cursor for cursor-based pagination",
			"schema": map[string]interface{}{
				"type":    "string",
				"example": "eyJpZCI6IjEyMyJ9",
			},
		},
	}
}

func mergeParameters(existing, additional []map[string]interface{}) []map[string]interface{} {
	if len(additional) == 0 {
		return existing
	}

	seen := map[string]struct{}{}
	for _, p := range existing {
		key, ok := parameterKey(p)
		if ok {
			seen[key] = struct{}{}
		}
	}

	out := append([]map[string]interface{}{}, existing...)
	for _, p := range additional {
		key, ok := parameterKey(p)
		if !ok {
			out = append(out, p)
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}

	return out
}

func parameterKey(p map[string]interface{}) (string, bool) {
	name, nameOK := p["name"].(string)
	in, inOK := p["in"].(string)
	if !nameOK || !inOK {
		return "", false
	}
	return strings.ToLower(in) + ":" + name, true
}

func applyPaginationResponse(operation map[string]interface{}, pagination *PaginationMetadata, extractor *SchemaExtractor, hasExplicit200 bool) {
	responses, ok := operation["responses"].(map[string]interface{})
	if !ok {
		return
	}
	if hasExplicit200 {
		return
	}

	itemSchema := map[string]interface{}{"type": "object"}
	if pagination != nil && pagination.ItemType != nil {
		itemSchema = extractor.ExtractSchemaRef(pagination.ItemType)
	}

	responses["200"] = map[string]interface{}{
		"description": "OK",
		"content": map[string]interface{}{
			web.ContentTypeApplicationJSON: map[string]interface{}{
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"items": map[string]interface{}{
							"type":  "array",
							"items": itemSchema,
						},
						"total": map[string]interface{}{
							"type":    "integer",
							"example": 100,
						},
						"next_cursor": map[string]interface{}{
							"type":     "string",
							"nullable": true,
							"example":  "eyJpZCI6IjEyMyJ9",
						},
					},
					"required": []string{"items", "total"},
				},
			},
		},
	}
}

func hasErrorDiagnostics(in []Diagnostic) bool {
	for _, d := range in {
		if normalizeDiagnosticSeverity(d.Severity) == "error" {
			return true
		}
	}
	return false
}

func diagnosticsToOpenAPI(in []Diagnostic) []map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(in))
	for _, d := range in {
		if strings.TrimSpace(d.Message) == "" {
			continue
		}
		out = append(out, map[string]interface{}{
			"severity": normalizeDiagnosticSeverity(d.Severity),
			"message":  d.Message,
		})
	}
	return out
}

func makeUniqueOperationID(base string, used map[string]struct{}) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "operation"
	}

	if _, exists := used[base]; !exists {
		used[base] = struct{}{}
		return base
	}

	for i := 2; ; i++ {
		candidate := base + "V" + strconv.Itoa(i)
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func applyGlobalOperationIDTagPrefix(operationID string, tags []string) string {
	enabled, sep := getOperationIDTagPrefixConfig()
	if !enabled || operationID == "" || len(tags) == 0 {
		return operationID
	}

	prefix := sanitizeOperationIDPart(tags[0])
	if prefix == "" {
		return operationID
	}
	if sep == "" {
		sep = "_"
	}

	candidate := prefix + sep + operationID
	if strings.HasPrefix(operationID, prefix+sep) {
		return operationID
	}
	return candidate
}

func generateDeterministicOperationID(method, path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	resourceParts := []string{}
	hasParam := false
	paramParts := []string{}

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, ":") || (strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")) {
			hasParam = true
			name := strings.Trim(seg, ":{}")
			if name != "" {
				paramParts = append(paramParts, toPascalCase(name))
			}
			continue
		}
		if isOperationIDIgnoredPathSegment(seg) {
			continue
		}
		resourceParts = append(resourceParts, singularize(seg))
	}

	resourcePart := toPascalCase(strings.Join(resourceParts, "_"))
	if resourcePart == "" {
		resourcePart = "Root"
	}

	var action string
	switch strings.ToUpper(method) {
	case "GET":
		if hasParam {
			action = "Get"
		} else {
			action = "List"
		}
	case "POST":
		action = "Create"
	case "PUT":
		action = "Update"
	case "PATCH":
		action = "Patch"
	case "DELETE":
		action = "Delete"
	case "HEAD":
		action = "Head"
	case "OPTIONS":
		action = "Options"
	case "TRACE":
		action = "Trace"
	case "CONNECT":
		action = "Connect"
	default:
		action = toPascalCase(strings.ToLower(method))
	}

	id := action + resourcePart
	if hasParam && len(paramParts) > 0 {
		id += "By" + strings.Join(paramParts, "And")
	}

	return toCamelCase(id)
}

func isOperationIDIgnoredPathSegment(seg string) bool {
	seg = strings.TrimSpace(strings.ToLower(seg))
	if seg == "" {
		return true
	}
	if seg == "api" {
		return true
	}
	if seg[0] == 'v' {
		allDigits := true
		for i := 1; i < len(seg); i++ {
			if seg[i] < '0' || seg[i] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits && len(seg) > 1 {
			return true
		}
	}
	return false
}

func toPascalCase(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})
	if len(parts) == 0 {
		parts = []string{s}
	}

	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
