package di

import (
	"fmt"
	"reflect"
	"sync"
)

// MetadataGroupName is the group tag for metadata entries.
const MetadataGroupName = "di:metadata"

// MetadataValue pairs a provided value with its metadata tag.
type MetadataValue struct {
	Value    any
	Type     reflect.Type
	Name     string
	Group    string
	Metadata any
}

type metadataKey struct {
	ptr uintptr
	typ reflect.Type
}

var metadataRegistry sync.Map

func metadataKeyFromValue(v any) (metadataKey, bool) {
	if v == nil {
		return metadataKey{}, false
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return metadataKey{}, false
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return metadataKey{}, false
	}
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Func, reflect.Chan, reflect.Slice, reflect.UnsafePointer:
		ptr := rv.Pointer()
		if ptr == 0 {
			return metadataKey{}, false
		}
		return metadataKey{ptr: ptr, typ: rv.Type()}, true
	default:
		return metadataKey{}, false
	}
}

func registerMetadata(value any, metadata []any) {
	if key, ok := metadataKeyFromValue(value); ok {
		metadataRegistry.Store(key, metadata)
	}
}

// RegisterMetadata attaches metadata to an instance programmatically.
// Note: only pointer-like values can be registered (same as ReflectMetadata).
func RegisterMetadata(value any, metadata ...any) {
	if len(metadata) == 0 {
		return
	}
	registerMetadata(value, metadata)
}

// ReflectMetadataAny returns metadata for a provided value, if registered.
func ReflectMetadataAny(value any) (any, bool) {
	key, ok := metadataKeyFromValue(value)
	if !ok {
		return nil, false
	}
	if raw, ok := metadataRegistry.Load(key); ok {
		return raw, true
	}
	return nil, false
}

// ReflectMetadata returns metadata for a provided value cast to T, if registered.
func ReflectMetadata[T any](value any) (T, bool) {
	var zero T
	raw, ok := ReflectMetadataAny(value)
	if !ok {
		return zero, false
	}
	casted, ok := raw.(T)
	if !ok {
		return zero, false
	}
	return casted, true
}

type metadataOption string

func (m metadataOption) applyBind(cfg *bindConfig) {
	if cfg.err != nil {
		return
	}
	value := string(m)
	if value == "" {
		cfg.err = fmt.Errorf(errMetadataNil)
		return
	}
	cfg.metadata = append(cfg.metadata, value)
}

func (m metadataOption) applyParam(*paramConfig) {}

// Metadata attaches a metadata tag to a provider.
func Metadata(value any) Option {
	if value == nil {
		return errorOption{err: fmt.Errorf(errMetadataNil)}
	}
	if s, ok := value.(string); ok {
		if s == "" {
			return errorOption{err: fmt.Errorf(errMetadataNil)}
		}
		return metadataOption(s)
	}
	return metadataAnyOption{value: value}
}

// MetadataGroup tags the next parameter to receive metadata entries.
func MetadataGroup() Option {
	return InGroup(MetadataGroupName)
}

type metadataAnyOption struct {
	value any
}

func (m metadataAnyOption) applyBind(cfg *bindConfig) {
	if cfg.err != nil {
		return
	}
	if m.value == nil {
		cfg.err = fmt.Errorf(errMetadataNil)
		return
	}
	cfg.metadata = append(cfg.metadata, m.value)
}

func (m metadataAnyOption) applyParam(*paramConfig) {}
