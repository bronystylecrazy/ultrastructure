package web

import (
	"github.com/gofiber/fiber/v3"
)

// Router extends fiber.Router with fluent API methods
type Router interface {
	// Standard HTTP methods that return RouteBuilder for chaining
	Get(path string, handlers ...fiber.Handler) *RouteBuilder
	Post(path string, handlers ...fiber.Handler) *RouteBuilder
	Put(path string, handlers ...fiber.Handler) *RouteBuilder
	Delete(path string, handlers ...fiber.Handler) *RouteBuilder
	Patch(path string, handlers ...fiber.Handler) *RouteBuilder
	Head(path string, handlers ...fiber.Handler) *RouteBuilder
	Options(path string, handlers ...fiber.Handler) *RouteBuilder
	All(path string, handlers ...fiber.Handler) Router

	// Group creates a sub-router with a prefix
	Group(prefix string, handlers ...fiber.Handler) Router

	// With returns a router that applies route options to every new route builder.
	With(opts ...RouteOption) Router

	// Tags sets default tags for all routes created from this router
	Tags(tags ...string) Router

	// Use adds middleware
	Use(args ...interface{}) Router
}

// routerWrapper wraps a fiber.Router and provides fluent API
type routerWrapper struct {
	fiberRouter   fiber.Router
	inheritedTags []string
	defaultOpts   []RouteOption
	registry      *MetadataRegistry
}

// NewRouterWithRegistry creates a router wrapper bound to a specific metadata registry.
func NewRouterWithRegistry(fiberRouter fiber.Router, registry *MetadataRegistry) Router {
	if registry == nil {
		registry = NewMetadataRegistry()
	}
	return &routerWrapper{
		fiberRouter: fiberRouter,
		registry:    registry,
	}
}

// Get registers a GET route and returns RouteBuilder for chaining
func (r *routerWrapper) Get(path string, handlers ...fiber.Handler) *RouteBuilder {
	return r.newBuilder("GET", path, handlers...)
}

// Post registers a POST route and returns RouteBuilder for chaining
func (r *routerWrapper) Post(path string, handlers ...fiber.Handler) *RouteBuilder {
	return r.newBuilder("POST", path, handlers...)
}

// Put registers a PUT route and returns RouteBuilder for chaining
func (r *routerWrapper) Put(path string, handlers ...fiber.Handler) *RouteBuilder {
	return r.newBuilder("PUT", path, handlers...)
}

// Delete registers a DELETE route and returns RouteBuilder for chaining
func (r *routerWrapper) Delete(path string, handlers ...fiber.Handler) *RouteBuilder {
	return r.newBuilder("DELETE", path, handlers...)
}

// Patch registers a PATCH route and returns RouteBuilder for chaining
func (r *routerWrapper) Patch(path string, handlers ...fiber.Handler) *RouteBuilder {
	return r.newBuilder("PATCH", path, handlers...)
}

// Head registers a HEAD route and returns RouteBuilder for chaining
func (r *routerWrapper) Head(path string, handlers ...fiber.Handler) *RouteBuilder {
	return r.newBuilder("HEAD", path, handlers...)
}

// Options registers an OPTIONS route and returns RouteBuilder for chaining
func (r *routerWrapper) Options(path string, handlers ...fiber.Handler) *RouteBuilder {
	return r.newBuilder("OPTIONS", path, handlers...)
}

// All registers a route for all HTTP methods.
func (r *routerWrapper) All(path string, handlers ...fiber.Handler) Router {
	if len(handlers) == 0 {
		r.fiberRouter.All(path, func(c fiber.Ctx) error { return c.Next() })
		return r
	}
	firstHandler := any(handlers[0])
	restHandlers := make([]any, len(handlers)-1)
	for i := 1; i < len(handlers); i++ {
		restHandlers[i-1] = handlers[i]
	}
	r.fiberRouter.All(path, firstHandler, restHandlers...)
	return r
}

// Group creates a new router group with prefix
func (r *routerWrapper) Group(prefix string, handlers ...fiber.Handler) Router {
	var groupRouter fiber.Router
	if len(handlers) == 0 {
		groupRouter = r.fiberRouter.Group(prefix)
	} else {
		allHandlers := make([]any, len(handlers))
		for i, h := range handlers {
			allHandlers[i] = h
		}
		groupRouter = r.fiberRouter.Group(prefix, allHandlers...)
	}
	// Inherit tags from parent router
	wrapper := NewRouterWithRegistry(groupRouter, r.registry).(*routerWrapper)
	wrapper.inheritedTags = append([]string{}, r.inheritedTags...)
	wrapper.defaultOpts = append([]RouteOption{}, r.defaultOpts...)
	return wrapper
}

// With returns a router that inherits this router and applies opts to every new route.
func (r *routerWrapper) With(opts ...RouteOption) Router {
	if len(opts) == 0 {
		return r
	}
	out := &routerWrapper{
		fiberRouter:   r.fiberRouter,
		registry:      r.registry,
		inheritedTags: append([]string{}, r.inheritedTags...),
		defaultOpts:   append([]RouteOption{}, r.defaultOpts...),
	}
	out.defaultOpts = append(out.defaultOpts, opts...)
	return out
}

// Tags sets default tags for all routes created from this router
func (r *routerWrapper) Tags(tags ...string) Router {
	r.inheritedTags = append(r.inheritedTags, tags...)
	return r
}

// Use adds middleware to the router
func (r *routerWrapper) Use(args ...interface{}) Router {
	r.fiberRouter.Use(args...)
	return r
}

func (r *routerWrapper) newBuilder(method, path string, handlers ...fiber.Handler) *RouteBuilder {
	b := newRouteBuilder(method, path, r.fiberRouter, r.registry, r.inheritedTags, handlers)
	if len(r.defaultOpts) > 0 {
		b.With(r.defaultOpts...)
	}
	return b
}
