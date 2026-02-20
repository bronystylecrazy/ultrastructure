package autoswag

import (
	"reflect"
	"sort"
	"sync"
)

// SwaggerModelRegistry stores extra model types that should be included
// in generated OpenAPI component schemas.
type SwaggerModelRegistry struct {
	mu     sync.RWMutex
	models map[reflect.Type]struct{}
}

// NewSwaggerModelRegistry creates a DI-friendly registry for additional swagger models.
func NewSwaggerModelRegistry() *SwaggerModelRegistry {
	return &SwaggerModelRegistry{
		models: make(map[reflect.Type]struct{}),
	}
}

// Add registers one or more model values or reflect.Type values.
// Nil values are ignored.
func (r *SwaggerModelRegistry) Add(models ...any) *SwaggerModelRegistry {
	if r == nil || len(models) == 0 {
		return r
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, model := range models {
		t := normalizeSwaggerModelInput(model)
		if t == nil {
			continue
		}
		r.models[t] = struct{}{}
	}

	return r
}

// AddNamed registers a model and forces its OpenAPI schema name.
// This works for both named and anonymous struct types.
func (r *SwaggerModelRegistry) AddNamed(name string, model any) *SwaggerModelRegistry {
	t := normalizeSwaggerModelInput(model)
	if t == nil {
		return r
	}

	registerSchemaNameForType(t, name)
	return r.Add(t)
}

// Types returns a deterministic snapshot of all registered model types.
func (r *SwaggerModelRegistry) Types() []reflect.Type {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]reflect.Type, 0, len(r.models))
	for t := range r.models {
		types = append(types, t)
	}

	sort.Slice(types, func(i, j int) bool {
		return types[i].String() < types[j].String()
	})

	return types
}

// Clear removes all registered model types.
func (r *SwaggerModelRegistry) Clear() {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.models = make(map[reflect.Type]struct{})
}

func normalizeSwaggerModelInput(model any) reflect.Type {
	if model == nil {
		return nil
	}
	if t, ok := model.(reflect.Type); ok {
		return normalizeSwaggerModelType(t)
	}
	return normalizeSwaggerModelType(reflect.TypeOf(model))
}

func normalizeSwaggerModelType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// AddSwaggerModelType registers model type T in the provided registry.
func AddSwaggerModelType[T any](r *SwaggerModelRegistry) {
	if r == nil {
		return
	}
	r.Add(reflect.TypeFor[T]())
}
