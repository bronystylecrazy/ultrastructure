package autoswag

import (
	"reflect"
	"sync"
)

type typeRulesRegistryState struct {
	mu      sync.RWMutex
	replace map[reflect.Type]reflect.Type
	skipped map[reflect.Type]struct{}
}

func newTypeRulesRegistryState() *typeRulesRegistryState {
	return &typeRulesRegistryState{
		replace: make(map[reflect.Type]reflect.Type),
		skipped: make(map[reflect.Type]struct{}),
	}
}

var typeRulesRegistry = newTypeRulesRegistryState()

// ReplaceType maps a source type to another type for OpenAPI generation.
// Example: ReplaceType[sql.NullInt64, int64]()
func ReplaceType[From any, To any]() {
	fromType := normalizeRuleType(reflect.TypeFor[From]())
	toType := normalizeRuleType(reflect.TypeFor[To]())
	if fromType == nil || toType == nil {
		return
	}

	typeRulesRegistry.mu.Lock()
	defer typeRulesRegistry.mu.Unlock()
	typeRulesRegistry.replace[fromType] = toType
}

// SkipType excludes type T from generated schema fields and parameters.
func SkipType[T any]() {
	var zero T
	t := normalizeRuleType(reflect.TypeOf(zero))
	if t == nil {
		return
	}

	typeRulesRegistry.mu.Lock()
	defer typeRulesRegistry.mu.Unlock()
	typeRulesRegistry.skipped[t] = struct{}{}
}

// ClearTypeRules clears all replace/skip rules.
// Primarily useful for tests.
func ClearTypeRules() {
	typeRulesRegistry.mu.Lock()
	defer typeRulesRegistry.mu.Unlock()
	typeRulesRegistry.replace = make(map[reflect.Type]reflect.Type)
	typeRulesRegistry.skipped = make(map[reflect.Type]struct{})
}

func normalizeRuleType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func resolveReplacedType(t reflect.Type) (reflect.Type, bool) {
	base := normalizeRuleType(t)
	if base == nil {
		return t, false
	}

	typeRulesRegistry.mu.RLock()
	defer typeRulesRegistry.mu.RUnlock()
	replaced, ok := typeRulesRegistry.replace[base]
	if !ok {
		return t, false
	}
	return replaced, true
}

func isSkippedType(t reflect.Type) bool {
	base := normalizeRuleType(t)
	if base == nil {
		return false
	}

	typeRulesRegistry.mu.RLock()
	defer typeRulesRegistry.mu.RUnlock()
	_, ok := typeRulesRegistry.skipped[base]
	return ok
}
