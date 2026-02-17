package web

import (
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// SwaggerContext provides mutable route metadata during OpenAPI generation.
type SwaggerContext struct {
	Route     RouteInfo
	Method    string
	Path      string
	Metadata  *RouteMetadata
	Models    *SwaggerModelRegistry
	Spec      *AutoSwaggerSpec
	Operation map[string]interface{}
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

func (c *SwaggerContext) AddParameter(parameter ParameterMetadata) {
	c.ensureMetadata()
	c.Metadata.Parameters = append(c.Metadata.Parameters, parameter)
}

func (c *SwaggerContext) SetRequestBody(model any, required bool, contentTypes ...string) {
	c.ensureMetadata()
	body := &RequestBodyMetadata{
		Type:     normalizeSwaggerModelInput(model),
		Required: required,
	}
	if len(contentTypes) > 0 {
		body.ContentTypes = append([]string(nil), contentTypes...)
	}
	c.Metadata.RequestBody = body
}

func (c *SwaggerContext) SetResponse(statusCode int, model any, description string) {
	c.ensureMetadata()
	if c.Metadata.Responses == nil {
		c.Metadata.Responses = make(map[int]ResponseMetadata)
	}
	responseType := normalizeSwaggerModelInput(model)
	contentType := ""
	if responseType != nil {
		contentType = "application/json"
	}
	c.Metadata.Responses[statusCode] = ResponseMetadata{
		Type:        responseType,
		ContentType: contentType,
		Description: description,
	}
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
