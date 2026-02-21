package autoswag

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/bronystylecrazy/ultrastructure/web"
)

// SwaggerContext provides mutable route metadata during OpenAPI generation.
type SwaggerContext struct {
	Route            RouteInfo
	Method           string
	Path             string
	Metadata         *RouteMetadata
	Models           *SwaggerModelRegistry
	Spec             *OpenAPISpec
	Operation        map[string]interface{}
	Warnings         []string
	Diagnostics      []Diagnostic
	conflictSeverity string
}

type Diagnostic struct {
	Severity string
	Message  string
}

// HookFunc mutates route metadata before operation generation.
type HookFunc func(ctx *SwaggerContext)

func (c *SwaggerContext) ensureMetadata() {
	if c != nil && c.Metadata == nil {
		c.Metadata = &RouteMetadata{}
	}
}

func (c *SwaggerContext) SetSummary(summary string) {
	c.ensureMetadata()
	c.Metadata.Summary = summary
}

func (c *SwaggerContext) SetDescription(description string) {
	c.ensureMetadata()
	c.Metadata.Description = description
}

func (c *SwaggerContext) SetOperationID(operationID string) {
	c.ensureMetadata()
	c.Metadata.OperationID = operationID
}

func (c *SwaggerContext) AddTag(tags ...string) {
	c.ensureMetadata()
	c.Metadata.Tags = append(c.Metadata.Tags, tags...)
}

func (c *SwaggerContext) AddTagDescription(name, description string) {
	c.ensureMetadata()
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" || description == "" {
		return
	}
	if c.Metadata.TagDescriptions == nil {
		c.Metadata.TagDescriptions = map[string]string{}
	}
	c.Metadata.TagDescriptions[name] = description
}

func (c *SwaggerContext) AddParameter(parameter ParameterMetadata) {
	c.ensureMetadata()
	c.Metadata.Parameters = append(c.Metadata.Parameters, parameter)
}

func (c *SwaggerContext) SetRequestBody(model any, required bool, contentTypes ...string) {
	c.ensureMetadata()
	existing := c.Metadata.RequestBody
	body := &RequestBodyMetadata{
		Type:     normalizeSwaggerModelInput(model),
		Required: required,
	}
	if len(contentTypes) > 0 {
		body.ContentTypes = append([]string(nil), contentTypes...)
	}
	body.Content = make(map[string]reflect.Type)
	if len(body.ContentTypes) == 0 {
		body.ContentTypes = []string{web.ContentTypeApplicationJSON}
	}
	for _, contentType := range body.ContentTypes {
		normalized := strings.TrimSpace(contentType)
		if normalized == "" {
			normalized = web.ContentTypeApplicationJSON
		}
		body.Content[normalized] = body.Type
	}
	if existing != nil && !requestBodySemanticallyEqual(existing, body) {
		c.addConflictf("requestBody conflict on %s %s: existing metadata differs from auto-detected payload", c.Method, c.Path)
	}
	c.Metadata.RequestBody = body
}

func (c *SwaggerContext) SetQuery(model any) {
	c.ensureMetadata()
	t := normalizeSwaggerModelInput(model)
	if c.Metadata.QueryType != nil && t != nil && c.Metadata.QueryType != t {
		c.addConflictf("query conflict on %s %s: existing type %s differs from auto-detected %s", c.Method, c.Path, c.Metadata.QueryType.String(), t.String())
	}
	c.Metadata.QueryType = t
}

func (c *SwaggerContext) SetResponse(statusCode int, model any, description string) {
	c.SetResponseAs(statusCode, model, web.ContentTypeApplicationJSON, description)
}

func (c *SwaggerContext) SetResponseAs(statusCode int, model any, contentType, description string) {
	c.ensureMetadata()
	if c.Metadata.Responses == nil {
		c.Metadata.Responses = make(map[int]ResponseMetadata)
	}
	responseType := normalizeSwaggerModelInput(model)
	contentType = strings.TrimSpace(contentType)
	if contentType == "" && responseType != nil {
		contentType = web.ContentTypeApplicationJSON
	}
	resp := c.Metadata.Responses[statusCode]
	if resp.Content == nil {
		resp.Content = make(map[string]reflect.Type)
	}
	if resp.ContentVariants == nil {
		resp.ContentVariants = make(map[string][]reflect.Type)
	}
	if existingType, ok := resp.Content[contentType]; ok && existingType != nil && responseType != nil && existingType != responseType {
		c.addConflictf(
			"response conflict on %s %s [%d %s]: existing type %s differs from auto-detected %s",
			c.Method,
			c.Path,
			statusCode,
			contentType,
			existingType.String(),
			responseType.String(),
		)
	}
	if contentType != "" {
		if existingType, ok := resp.Content[contentType]; ok {
			resp.ContentVariants[contentType] = appendUniqueReflectType(resp.ContentVariants[contentType], existingType)
		}
		resp.ContentVariants[contentType] = appendUniqueReflectType(resp.ContentVariants[contentType], responseType)
		resp.Content[contentType] = responseType
	}
	resp.Type = responseType
	resp.ContentType = contentType
	resp.NoContent = statusCode == 204
	if strings.TrimSpace(description) != "" {
		if strings.TrimSpace(resp.Description) != "" && strings.TrimSpace(resp.Description) != strings.TrimSpace(description) {
			c.addConflictf(
				"response description conflict on %s %s [%d]: existing=%q auto-detected=%q",
				c.Method,
				c.Path,
				statusCode,
				resp.Description,
				description,
			)
		}
		resp.Description = description
	}
	c.Metadata.Responses[statusCode] = resp
}

func (c *SwaggerContext) AddResponseHeader(statusCode int, name string, model any, description string) {
	c.ensureMetadata()
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if c.Metadata.Responses == nil {
		c.Metadata.Responses = make(map[int]ResponseMetadata)
	}

	resp := c.Metadata.Responses[statusCode]
	if resp.Headers == nil {
		resp.Headers = make(map[string]ResponseHeaderMetadata)
	}

	headerType := normalizeSwaggerModelInput(model)
	if existing, ok := resp.Headers[name]; ok && existing.Type != nil && headerType != nil && existing.Type != headerType {
		c.addConflictf(
			"response header conflict on %s %s [%d %s]: existing type %s differs from auto-detected %s",
			c.Method,
			c.Path,
			statusCode,
			name,
			existing.Type.String(),
			headerType.String(),
		)
	}

	header := resp.Headers[name]
	header.Type = headerType
	if strings.TrimSpace(description) != "" {
		header.Description = strings.TrimSpace(description)
	}
	resp.Headers[name] = header
	c.Metadata.Responses[statusCode] = resp
}

func (c *SwaggerContext) SetOperationField(key string, value any) {
	if c == nil || key == "" {
		return
	}
	if c.Operation == nil {
		c.Operation = make(map[string]interface{})
	}
	c.Operation[key] = value
}

func (c *SwaggerContext) addWarningf(format string, args ...any) {
	c.addDiagnosticf("warning", format, args...)
}

func (c *SwaggerContext) addConflictf(format string, args ...any) {
	severity := normalizeDiagnosticSeverity(c.conflictSeverity)
	c.addDiagnosticf(severity, format, args...)
}

func (c *SwaggerContext) addDiagnosticf(severity, format string, args ...any) {
	if c == nil {
		return
	}
	severity = normalizeDiagnosticSeverity(severity)
	msg := strings.TrimSpace(fmt.Sprintf(format, args...))
	if msg == "" {
		return
	}
	c.Warnings = append(c.Warnings, msg)
	c.Diagnostics = append(c.Diagnostics, Diagnostic{
		Severity: severity,
		Message:  msg,
	})
}

// RouteModelType returns a stable "primary" model type for this route context.
// Priority: request body type, then lowest-status response type, then query type,
// then parameter types, then pagination item type.
func (c *SwaggerContext) RouteModelType() reflect.Type {
	if c == nil || c.Metadata == nil {
		return nil
	}

	if c.Metadata.RequestBody != nil && c.Metadata.RequestBody.Type != nil {
		return normalizeSwaggerModelType(c.Metadata.RequestBody.Type)
	}

	if len(c.Metadata.Responses) > 0 {
		codes := make([]int, 0, len(c.Metadata.Responses))
		for code := range c.Metadata.Responses {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		for _, code := range codes {
			if t := normalizeSwaggerModelType(c.Metadata.Responses[code].Type); t != nil {
				return t
			}
		}
	}

	if c.Metadata.QueryType != nil {
		return normalizeSwaggerModelType(c.Metadata.QueryType)
	}

	for _, p := range c.Metadata.Parameters {
		if t := normalizeSwaggerModelType(p.Type); t != nil {
			return t
		}
	}

	if c.Metadata.Pagination != nil && c.Metadata.Pagination.ItemType != nil {
		return normalizeSwaggerModelType(c.Metadata.Pagination.ItemType)
	}

	return nil
}

// RouteModelPackagePath returns the import path of the primary model type.
func (c *SwaggerContext) RouteModelPackagePath() string {
	t := c.RouteModelType()
	if t == nil {
		return ""
	}
	return strings.TrimSpace(t.PkgPath())
}

// RouteModelPackageName returns the last path segment of the primary model package.
func (c *SwaggerContext) RouteModelPackageName() string {
	pkgPath := c.RouteModelPackagePath()
	if pkgPath == "" {
		return ""
	}
	return path.Base(pkgPath)
}

type hookRegistryState struct {
	mu   sync.RWMutex
	hook HookFunc
}

func newHookRegistryState() *hookRegistryState {
	return &hookRegistryState{}
}

var hookRegistry = newHookRegistryState()

// RegisterGlobalHook registers a global metadata hook used during OpenAPI generation.
// The hook can mutate any RouteMetadata fields (operationId, tags, request body, responses, etc).
func RegisterGlobalHook(hook HookFunc) {
	hookRegistry.mu.Lock()
	defer hookRegistry.mu.Unlock()
	hookRegistry.hook = hook
}

// ClearGlobalHook removes the registered global metadata hook.
func ClearGlobalHook() {
	hookRegistry.mu.Lock()
	defer hookRegistry.mu.Unlock()
	hookRegistry.hook = nil
}

func getRegisteredHook() HookFunc {
	hookRegistry.mu.RLock()
	defer hookRegistry.mu.RUnlock()
	return hookRegistry.hook
}

func requestBodySemanticallyEqual(a, b *RequestBodyMetadata) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Type != b.Type || a.Required != b.Required || a.RequireAtLeastOne != b.RequireAtLeastOne {
		return false
	}
	return reflect.DeepEqual(normalizeRequestBodyContentMap(a.Content), normalizeRequestBodyContentMap(b.Content))
}

func normalizeRequestBodyContentMap(content map[string]reflect.Type) map[string]reflect.Type {
	if len(content) == 0 {
		return map[string]reflect.Type{}
	}
	out := make(map[string]reflect.Type, len(content))
	for k, v := range content {
		ct := strings.TrimSpace(k)
		if ct == "" {
			ct = web.ContentTypeApplicationJSON
		}
		out[ct] = v
	}
	return out
}

func appendUniqueReflectType(existing []reflect.Type, t reflect.Type) []reflect.Type {
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

func normalizeDiagnosticSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error":
		return "error"
	default:
		return "warning"
	}
}
