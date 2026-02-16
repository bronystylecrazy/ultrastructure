package web

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func parseTagValue(raw string, t reflect.Type) (interface{}, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || t == nil {
		return nil, false
	}

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t == reflect.TypeOf(time.Time{}) || t == reflect.TypeOf(uuid.UUID{}) {
		return raw, true
	}

	switch t.Kind() {
	case reflect.String:
		return raw, true
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, false
		}
		return v, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, false
		}
		return v, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, false
		}
		return v, true
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, false
		}
		return v, true
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		var out interface{}
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return nil, false
		}
		return out, true
	default:
		return raw, true
	}
}
