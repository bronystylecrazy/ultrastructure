package web

import (
	"reflect"
	"sync"
)

type enumRegistryState struct {
	mu     sync.RWMutex
	values map[reflect.Type][]interface{}
}

func newEnumRegistryState() *enumRegistryState {
	return &enumRegistryState{
		values: make(map[reflect.Type][]interface{}),
	}
}

var enumRegistry = newEnumRegistryState()

// RegisterEnum registers enum values for a Go type so schemas can emit OpenAPI enum constraints.
// Usage: RegisterEnum(StatusActive, StatusInactive)
func RegisterEnum[T any](values ...T) {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil || len(values) == 0 {
		return
	}

	enumValues := make([]interface{}, 0, len(values))
	for _, v := range values {
		enumValues = append(enumValues, v)
	}

	enumRegistry.mu.Lock()
	defer enumRegistry.mu.Unlock()
	enumRegistry.values[t] = enumValues
}

// ClearEnumRegistry removes all registered enum values.
// Primarily useful for tests.
func ClearEnumRegistry() {
	enumRegistry.mu.Lock()
	defer enumRegistry.mu.Unlock()
	enumRegistry.values = make(map[reflect.Type][]interface{})
}

func getRegisteredEnumValues(t reflect.Type) ([]interface{}, bool) {
	if t == nil {
		return nil, false
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	enumRegistry.mu.RLock()
	defer enumRegistry.mu.RUnlock()
	values, ok := enumRegistry.values[t]
	if !ok || len(values) == 0 {
		return nil, false
	}

	cloned := append([]interface{}(nil), values...)
	return cloned, true
}
