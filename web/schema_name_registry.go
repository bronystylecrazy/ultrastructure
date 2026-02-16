package web

import (
	"reflect"
	"strings"
	"sync"
)

var schemaNameRegistry = struct {
	mu    sync.RWMutex
	names map[reflect.Type]string
}{
	names: make(map[reflect.Type]string),
}

// RegisterSchemaName overrides the OpenAPI component schema name for type T.
// Usage: RegisterSchemaName[MyType]("User")
func RegisterSchemaName[T any](name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}

	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schemaNameRegistry.mu.Lock()
	defer schemaNameRegistry.mu.Unlock()
	schemaNameRegistry.names[t] = name
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
