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

	// Group creates a sub-router with a prefix
	Group(prefix string, handlers ...fiber.Handler) Router

	// Tags sets default tags for all routes created from this router
	Tags(tags ...string) Router

	// Use adds middleware
	Use(args ...interface{}) Router
}

// routerWrapper wraps a fiber.Router and provides fluent API
type routerWrapper struct {
	fiberRouter   fiber.Router
	inheritedTags []string
}

// NewRouter creates a new Router wrapper
func NewRouter(fiberRouter fiber.Router) Router {
	return &routerWrapper{
		fiberRouter: fiberRouter,
	}
}

// Get registers a GET route and returns RouteBuilder for chaining
func (r *routerWrapper) Get(path string, handlers ...fiber.Handler) *RouteBuilder {
	return newRouteBuilder("GET", path, r.fiberRouter, r.inheritedTags, handlers)
}

// Post registers a POST route and returns RouteBuilder for chaining
func (r *routerWrapper) Post(path string, handlers ...fiber.Handler) *RouteBuilder {
	return newRouteBuilder("POST", path, r.fiberRouter, r.inheritedTags, handlers)
}

// Put registers a PUT route and returns RouteBuilder for chaining
func (r *routerWrapper) Put(path string, handlers ...fiber.Handler) *RouteBuilder {
	return newRouteBuilder("PUT", path, r.fiberRouter, r.inheritedTags, handlers)
}

// Delete registers a DELETE route and returns RouteBuilder for chaining
func (r *routerWrapper) Delete(path string, handlers ...fiber.Handler) *RouteBuilder {
	return newRouteBuilder("DELETE", path, r.fiberRouter, r.inheritedTags, handlers)
}

// Patch registers a PATCH route and returns RouteBuilder for chaining
func (r *routerWrapper) Patch(path string, handlers ...fiber.Handler) *RouteBuilder {
	return newRouteBuilder("PATCH", path, r.fiberRouter, r.inheritedTags, handlers)
}

// Head registers a HEAD route and returns RouteBuilder for chaining
func (r *routerWrapper) Head(path string, handlers ...fiber.Handler) *RouteBuilder {
	return newRouteBuilder("HEAD", path, r.fiberRouter, r.inheritedTags, handlers)
}

// Options registers an OPTIONS route and returns RouteBuilder for chaining
func (r *routerWrapper) Options(path string, handlers ...fiber.Handler) *RouteBuilder {
	return newRouteBuilder("OPTIONS", path, r.fiberRouter, r.inheritedTags, handlers)
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
	wrapper := NewRouter(groupRouter).(*routerWrapper)
	wrapper.inheritedTags = append([]string{}, r.inheritedTags...)
	return wrapper
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
