package web

import (
	"reflect"
	"strings"
	"sync"
)

type schemaNameRegistryState struct {
	mu    sync.RWMutex
	names map[reflect.Type]string
}

func newSchemaNameRegistryState() *schemaNameRegistryState {
	return &schemaNameRegistryState{
		names: make(map[reflect.Type]string),
	}
}

var schemaNameRegistry = newSchemaNameRegistryState()

// RegisterSchemaName overrides the OpenAPI component schema name for type T.
// Usage: RegisterSchemaName[MyType]("User")
func RegisterSchemaName[T any](name string) {
	var zero T
	registerSchemaNameForType(reflect.TypeOf(zero), name)
}

// RegisterSchemaNameForModel overrides the OpenAPI component schema name for a model value/type.
// model can be a value, pointer, or reflect.Type.
func RegisterSchemaNameForModel(model any, name string) {
	registerSchemaNameForType(normalizeSwaggerModelInput(model), name)
}

// ClearSchemaNameRegistry removes all schema name overrides.
// Primarily useful for tests.
func ClearSchemaNameRegistry() {
	schemaNameRegistry.mu.Lock()
	defer schemaNameRegistry.mu.Unlock()
	schemaNameRegistry.names = make(map[reflect.Type]string)
}

func getRegisteredSchemaName(t reflect.Type) (string, bool) {
	if t == nil {
		return "", false
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schemaNameRegistry.mu.RLock()
	defer schemaNameRegistry.mu.RUnlock()
	name, ok := schemaNameRegistry.names[t]
	if !ok || strings.TrimSpace(name) == "" {
		return "", false
	}
	return name, true
}

func registerSchemaNameForType(t reflect.Type, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}

	t = normalizeSwaggerModelType(t)
	if t == nil {
		return
	}

	schemaNameRegistry.mu.Lock()
	defer schemaNameRegistry.mu.Unlock()
	schemaNameRegistry.names[t] = name
}
