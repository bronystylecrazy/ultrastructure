package web

import (
	"reflect"
	"strings"
	"sync"
)

// RouteMetadata stores OpenAPI metadata for a single route
type RouteMetadata struct {
	OperationID string
	Tags        []string
	TagDescriptions map[string]string
	Summary     string
	Description string
	RequestBody *RequestBodyMetadata
	QueryType   reflect.Type
	Parameters  []ParameterMetadata
	Security    []SecurityRequirement
	Policies    []string
	Pagination  *PaginationMetadata
	Responses   map[int]ResponseMetadata // statusCode -> metadata
	Examples    map[int]interface{}      // statusCode -> example
}

// ParameterMetadata stores header/cookie/query parameter schema metadata.
type ParameterMetadata struct {
	Name        string
	In          string // query, header, cookie
	Type        reflect.Type
	Required    bool
	Description string
	Extensions  string // e.g. "x-nullable,x-owner=team,!x-omitempty"
}

// SecurityRequirement stores OpenAPI operation security requirements.
type SecurityRequirement struct {
	Scheme string
	Scopes []string
}

// PaginationMetadata stores automatic pagination documentation settings.
type PaginationMetadata struct {
	ItemType reflect.Type
}

// RequestBodyMetadata stores request body schema metadata.
type RequestBodyMetadata struct {
	Type              reflect.Type
	Required          bool
	ContentTypes      []string
	Content           map[string]reflect.Type
	RequireAtLeastOne bool
}

// ResponseMetadata stores response schema metadata for a status code.
type ResponseMetadata struct {
	Type        reflect.Type
	ContentType string
	Content     map[string]reflect.Type
	// ContentVariants stores multiple possible model types for a media type.
	// This is rendered as OpenAPI oneOf when more than one type exists.
	ContentVariants map[string][]reflect.Type
	NoContent   bool
	Description string
	Headers     map[string]ResponseHeaderMetadata
}

// ResponseHeaderMetadata stores OpenAPI response header schema metadata.
type ResponseHeaderMetadata struct {
	Type        reflect.Type
	Description string
}

// MetadataRegistry stores metadata for all routes
type MetadataRegistry struct {
	mu     sync.RWMutex
	routes map[string]*RouteMetadata // "METHOD:path" -> metadata
}

// NewMetadataRegistry creates an isolated route metadata registry.
func NewMetadataRegistry() *MetadataRegistry {
	return &MetadataRegistry{
		routes: make(map[string]*RouteMetadata),
	}
}

// RegisterRoute stores metadata for a route
func (r *MetadataRegistry) RegisterRoute(method, path string, metadata *RouteMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.ToUpper(strings.TrimSpace(method)) + ":" + normalizeRegistryPath(path)
	r.routes[key] = metadata
}

// GetRoute retrieves metadata for a route
func (r *MetadataRegistry) GetRoute(method, path string) *RouteMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToUpper(strings.TrimSpace(method)) + ":" + normalizeRegistryPath(path)
	return r.routes[key]
}

// AllRoutes returns all registered route metadata
func (r *MetadataRegistry) AllRoutes() map[string]*RouteMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	routes := make(map[string]*RouteMetadata, len(r.routes))
	for k, v := range r.routes {
		routes[k] = v
	}
	return routes
}

// Clear removes all registered routes (useful for testing)
func (r *MetadataRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes = make(map[string]*RouteMetadata)
}

func normalizeRegistryPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	if path == "" {
		return "/"
	}
	return path
}
