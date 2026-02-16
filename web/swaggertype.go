package web

import "strings"

func schemaFromSwaggerTypeTag(tag string) (map[string]interface{}, bool) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, false
	}

	parts := strings.Split(tag, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) == 1 {
		if schema := primitiveSchemaByName(parts[0]); schema != nil {
			return schema, true
		}
		return nil, false
	}

	if len(parts) == 2 && parts[0] == "primitive" {
		if schema := primitiveSchemaByName(parts[1]); schema != nil {
			return schema, true
		}
		return nil, false
	}

	if len(parts) == 2 && parts[0] == "array" {
		if item := primitiveSchemaByName(parts[1]); item != nil {
			return map[string]interface{}{
				"type":  "array",
				"items": item,
			}, true
		}
		return nil, false
	}

	return nil, false
}

func primitiveSchemaByName(name string) map[string]interface{} {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "string":
		return map[string]interface{}{"type": "string"}
	case "boolean", "bool":
		return map[string]interface{}{"type": "boolean"}
	case "integer", "int":
		return map[string]interface{}{"type": "integer"}
	case "number", "float":
		return map[string]interface{}{"type": "number"}
	case "file", "binary":
		return map[string]interface{}{"type": "string", "format": "binary"}
	case "object":
		return map[string]interface{}{"type": "object"}
	default:
		return nil
	}
}

func exampleFromSchemaType(schema map[string]interface{}) interface{} {
	if schema == nil {
		return nil
	}

	t, _ := schema["type"].(string)
	switch t {
	case "string":
		return "example string"
	case "integer":
		return 42
	case "number":
		return 3.14
	case "boolean":
		return true
	case "array":
		if item, ok := schema["items"].(map[string]interface{}); ok {
			return []interface{}{exampleFromSchemaType(item)}
		}
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return nil
	}
}
