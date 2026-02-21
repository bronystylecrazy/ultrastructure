package autoswag

import (
	"reflect"
	"strings"
	"sync"
)

type fieldDescriptionRegistryState struct {
	mu    sync.RWMutex
	items map[reflect.Type]map[string]string
}

func newFieldDescriptionRegistryState() *fieldDescriptionRegistryState {
	return &fieldDescriptionRegistryState{
		items: make(map[reflect.Type]map[string]string),
	}
}

var fieldDescriptionRegistry = newFieldDescriptionRegistryState()

func RegisterFieldDescription(model any, fieldName, description string) {
	t := normalizeSwaggerModelInput(model)
	fieldName = strings.TrimSpace(fieldName)
	description = strings.TrimSpace(description)
	if t == nil || fieldName == "" || description == "" {
		return
	}
	fieldDescriptionRegistry.mu.Lock()
	defer fieldDescriptionRegistry.mu.Unlock()
	if fieldDescriptionRegistry.items[t] == nil {
		fieldDescriptionRegistry.items[t] = map[string]string{}
	}
	fieldDescriptionRegistry.items[t][fieldName] = description
}

func getRegisteredFieldDescription(t reflect.Type, fieldName string) (string, bool) {
	if t == nil {
		return "", false
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fieldName = strings.TrimSpace(fieldName)
	if fieldName == "" {
		return "", false
	}
	fieldDescriptionRegistry.mu.RLock()
	defer fieldDescriptionRegistry.mu.RUnlock()
	fields := fieldDescriptionRegistry.items[t]
	if len(fields) == 0 {
		return "", false
	}
	v, ok := fields[fieldName]
	return strings.TrimSpace(v), ok
}

func clearFieldDescriptionRegistry() {
	fieldDescriptionRegistry.mu.Lock()
	defer fieldDescriptionRegistry.mu.Unlock()
	fieldDescriptionRegistry.items = make(map[reflect.Type]map[string]string)
}
