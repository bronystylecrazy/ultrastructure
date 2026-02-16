package web

import (
	"reflect"
	"sync"
)

// RouteMetadata stores OpenAPI metadata for a single route
type RouteMetadata struct {
	OperationID string
	Tags        []string
	Summary     string
	Description string
	RequestBody *RequestBodyMetadata
	QueryType   reflect.Type
	Parameters  []ParameterMetadata
	Security    []SecurityRequirement
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
	RequireAtLeastOne bool
}

// ResponseMetadata stores response schema metadata for a status code.
type ResponseMetadata struct {
	Type        reflect.Type
	ContentType string
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

// Global registry instance
var globalRegistry = &MetadataRegistry{
	routes: make(map[string]*RouteMetadata),
}

// GetGlobalRegistry returns the global metadata registry
func GetGlobalRegistry() *MetadataRegistry {
	return globalRegistry
}

// RegisterRoute stores metadata for a route
func (r *MetadataRegistry) RegisterRoute(method, path string, metadata *RouteMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := method + ":" + path
	r.routes[key] = metadata
}

// GetRoute retrieves metadata for a route
func (r *MetadataRegistry) GetRoute(method, path string) *RouteMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := method + ":" + path
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
