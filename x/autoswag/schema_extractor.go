package autoswag

import (
	"fmt"
	"mime/multipart"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// SchemaExtractor extracts OpenAPI schemas from Go types using reflection
type SchemaExtractor struct {
	schemas      map[string]interface{}
	typeNames    map[reflect.Type]string
	claimedNames map[string]reflect.Type
}

// NewSchemaExtractor creates a new schema extractor
func NewSchemaExtractor() *SchemaExtractor {
	return &SchemaExtractor{
		schemas:      make(map[string]interface{}),
		typeNames:    make(map[reflect.Type]string),
		claimedNames: make(map[string]reflect.Type),
	}
}

// ExtractSchema analyzes a Go type and returns an OpenAPI schema
func (e *SchemaExtractor) ExtractSchema(t reflect.Type) map[string]interface{} {
	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if we already have this schema
	typeName := e.getTypeName(t)
	if _, exists := e.schemas[typeName]; exists {
		return map[string]interface{}{
			"$ref": fmt.Sprintf("#/components/schemas/%s", typeName),
		}
	}

	// Extract schema based on type
	extractedSchema := e.extractTypeSchema(t)

	// Store in schemas registry if it's a struct
	if t.Kind() == reflect.Struct && typeName != "" {
		e.schemas[typeName] = extractedSchema
	}

	return extractedSchema
}

// ExtractSchemaRef analyzes a Go type and returns a schema reference for
// named struct types while ensuring component schemas are registered.
func (e *SchemaExtractor) ExtractSchemaRef(t reflect.Type) map[string]interface{} {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if !isRefEligibleType(t) {
		return e.ExtractSchema(t)
	}

	typeName := e.getTypeName(t)
	if typeName == "" {
		return e.ExtractSchema(t)
	}

	if _, exists := e.schemas[typeName]; !exists {
		e.schemas[typeName] = e.extractTypeSchema(t)
	}

	return map[string]interface{}{
		"$ref": fmt.Sprintf("#/components/schemas/%s", typeName),
	}
}

func isRefEligibleType(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	// Struct-backed scalar types should stay inline primitives.
	if t == reflect.TypeOf(time.Time{}) || t == reflect.TypeOf(uuid.UUID{}) {
		return false
	}
	return true
}

// extractTypeSchema extracts schema for a specific type
func (e *SchemaExtractor) extractTypeSchema(t reflect.Type) map[string]interface{} {
	originalType := t
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if replaced, ok := resolveReplacedType(t); ok {
		t = replaced
	}

	// Special-case known struct-backed scalar types before kind switching.
	switch t {
	case reflect.TypeOf(time.Time{}):
		schema := map[string]interface{}{
			"type":    "string",
			"format":  "date-time",
			"example": "2024-01-01T00:00:00Z",
		}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	case reflect.TypeOf(uuid.UUID{}):
		schema := map[string]interface{}{
			"type":    "string",
			"format":  "uuid",
			"example": "123e4567-e89b-12d3-a456-426614174000",
		}
		applyRegisteredEnumToSchema(schema, originalType)
		return schema
	case reflect.TypeOf(multipart.FileHeader{}):
		return map[string]interface{}{
			"type":   "string",
			"format": "binary",
		}
	}

	var schema map[string]interface{}
	switch t.Kind() {
	case reflect.Struct:
		schema = e.extractStructSchema(t)
	case reflect.Slice, reflect.Array:
		schema = e.extractArraySchema(t)
	case reflect.Map:
		schema = e.extractMapSchema(t)
	case reflect.String:
		schema = e.extractStringSchema(t)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema = map[string]interface{}{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema = map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		schema = map[string]interface{}{"type": "number"}
	case reflect.Bool:
		schema = map[string]interface{}{"type": "boolean"}
	default:
		schema = map[string]interface{}{"type": "string"}
	}

	applyRegisteredEnumToSchema(schema, originalType)
	return schema
}

func applyRegisteredEnumToSchema(schema map[string]interface{}, t reflect.Type) {
	if schema == nil {
		return
	}
	if _, hasEnum := schema["enum"]; hasEnum {
		return
	}
	if values, ok := getRegisteredEnumValues(t); ok {
		schema["enum"] = values
	}
}

// extractStructSchema extracts schema from a struct type
func (e *SchemaExtractor) extractStructSchema(t reflect.Type) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}
	example := make(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}
		if shouldSwaggerIgnoreField(field) {
			continue
		}
		if isSkippedType(field.Type) {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Parse JSON tag
		jsonName := field.Name
		omitempty := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				jsonName = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitempty = true
				}
			}
		}

		// Extract field schema
		fieldSchema := e.extractTypeSchema(field.Type)
		if overrideSchema, ok := schemaFromSwaggerTypeTag(field.Tag.Get("swaggertype")); ok {
			fieldSchema = overrideSchema
		}
		if field.Type.Kind() == reflect.Ptr {
			fieldSchema["nullable"] = true
		}

		// Add description from struct tags if available
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema["description"] = desc
		} else if desc, ok := getRegisteredFieldDescription(t, field.Name); ok && strings.TrimSpace(desc) != "" {
			fieldSchema["description"] = desc
		}

		applyValidationTags(fieldSchema, field.Type, field.Tag.Get("validate"))
		if value, ok := parseTagValue(field.Tag.Get("example"), field.Type); ok {
			fieldSchema["example"] = value
		}
		if value, ok := parseTagValue(field.Tag.Get("default"), field.Type); ok {
			fieldSchema["default"] = value
		}
		applyExtensionsTag(fieldSchema, field.Tag.Get("extensions"))

		properties[jsonName] = fieldSchema

		// Add to required based on tags/type inference.
		if isSchemaFieldRequired(field, omitempty) {
			required = append(required, jsonName)
		}

		// Generate example value (explicit example tag takes precedence).
		if value, ok := parseTagValue(field.Tag.Get("example"), field.Type); ok {
			example[jsonName] = value
		} else if overrideSchema, ok := schemaFromSwaggerTypeTag(field.Tag.Get("swaggertype")); ok {
			example[jsonName] = exampleFromSchemaType(overrideSchema)
		} else if replacedType, ok := resolveReplacedType(field.Type); ok {
			example[jsonName] = e.generateExampleValue(replacedType)
		} else {
			example[jsonName] = e.generateExampleValue(field.Type)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	if len(example) > 0 {
		schema["example"] = example
	}

	return schema
}

// extractArraySchema extracts schema from array/slice type
func (e *SchemaExtractor) extractArraySchema(t reflect.Type) map[string]interface{} {
	elemType := t.Elem()
	return map[string]interface{}{
		"type":  "array",
		"items": e.extractTypeSchema(elemType),
	}
}

// extractMapSchema extracts schema from map type
func (e *SchemaExtractor) extractMapSchema(t reflect.Type) map[string]interface{} {
	return map[string]interface{}{
		"type":                 "object",
		"additionalProperties": e.extractTypeSchema(t.Elem()),
	}
}

// extractStringSchema handles special string types (UUID, time, etc.)
func (e *SchemaExtractor) extractStringSchema(t reflect.Type) map[string]interface{} {
	return map[string]interface{}{
		"type": "string",
	}
}

// generateExampleValue generates example values for different types
func (e *SchemaExtractor) generateExampleValue(t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t {
	case reflect.TypeOf(time.Time{}):
		return "2024-01-01T00:00:00Z"
	case reflect.TypeOf(uuid.UUID{}):
		return "123e4567-e89b-12d3-a456-426614174000"
	}

	switch t.Kind() {
	case reflect.String:
		return "example string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 42
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 42
	case reflect.Float32, reflect.Float64:
		return 3.14
	case reflect.Bool:
		return true
	case reflect.Slice, reflect.Array:
		return []interface{}{e.generateExampleValue(t.Elem())}
	case reflect.Map:
		return map[string]interface{}{
			"key": e.generateExampleValue(t.Elem()),
		}
	case reflect.Struct:
		example := make(map[string]interface{})
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			if shouldSwaggerIgnoreField(field) {
				continue
			}
			if isSkippedType(field.Type) {
				continue
			}
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}
			jsonName := field.Name
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					jsonName = parts[0]
				}
			}
			example[jsonName] = e.generateExampleValue(field.Type)
		}
		return example
	default:
		return nil
	}
}

func shouldSwaggerIgnoreField(field reflect.StructField) bool {
	v := strings.TrimSpace(strings.ToLower(field.Tag.Get("swaggerignore")))
	return v == "true"
}

// getTypeName returns a human-readable name for a type
func (e *SchemaExtractor) getTypeName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if registeredName, ok := getRegisteredSchemaName(t); ok {
		return registeredName
	}

	// Use the type name if available.
	if t.Name() == "" {
		return ""
	}

	if existing, ok := e.typeNames[t]; ok {
		return existing
	}

	base := t.Name()
	pkgSegment := sanitizePackageSegment(t.PkgPath())
	candidate := base
	if pkgSegment != "" {
		candidate = pkgSegment + "." + base
	}
	if owner, exists := e.claimedNames[candidate]; exists && owner != t {
		baseCandidate := candidate
		if baseCandidate == "" {
			baseCandidate = "Type"
		}
		suffix := 2
		for {
			candidate = fmt.Sprintf("%s_%d", baseCandidate, suffix)
			if owner, exists := e.claimedNames[candidate]; !exists || owner == t {
				break
			}
			suffix++
		}
	}

	e.typeNames[t] = candidate
	e.claimedNames[candidate] = t
	return candidate
}

// GetSchemas returns all registered schemas
func (e *SchemaExtractor) GetSchemas() map[string]interface{} {
	return e.schemas
}

func sanitizePackageSegment(pkgPath string) string {
	segment := path.Base(strings.TrimSpace(pkgPath))
	if segment == "" || segment == "." || segment == "/" {
		return ""
	}
	var b strings.Builder
	for _, r := range segment {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return strings.Trim(b.String(), "_")
}

func isSchemaFieldRequired(field reflect.StructField, omitempty bool) bool {
	if hasSchemaValidateRequired(field.Tag.Get("validate")) {
		return true
	}

	if omitempty {
		return false
	}

	if field.Type.Kind() == reflect.Ptr {
		return false
	}

	return true
}

func hasSchemaValidateRequired(validateTag string) bool {
	for _, p := range strings.Split(validateTag, ",") {
		if strings.TrimSpace(p) == "required" {
			return true
		}
	}
	return false
}

func applyValidationTags(schema map[string]interface{}, fieldType reflect.Type, validateTag string) {
	if validateTag == "" {
		return
	}

	t := fieldType
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for _, token := range strings.Split(validateTag, ",") {
		token = strings.TrimSpace(token)
		if token == "" || token == "required" {
			continue
		}

		switch {
		case token == "email" && schema["type"] == "string":
			schema["format"] = "email"
		case token == "url" && schema["type"] == "string":
			schema["format"] = "uri"
		case token == "uuid" && schema["type"] == "string":
			schema["format"] = "uuid"
		case strings.HasPrefix(token, "oneof="):
			values := strings.Fields(strings.TrimPrefix(token, "oneof="))
			if len(values) > 0 {
				schema["enum"] = parseEnumValues(values, t)
			}
		case strings.HasPrefix(token, "len="):
			v, ok := parseNumber(strings.TrimPrefix(token, "len="))
			if ok {
				applyLenConstraint(schema, v)
			}
		case strings.HasPrefix(token, "min="):
			v, ok := parseNumber(strings.TrimPrefix(token, "min="))
			if ok {
				applyMinConstraint(schema, v)
			}
		case strings.HasPrefix(token, "max="):
			v, ok := parseNumber(strings.TrimPrefix(token, "max="))
			if ok {
				applyMaxConstraint(schema, v)
			}
		case strings.HasPrefix(token, "gt="):
			v, ok := parseNumber(strings.TrimPrefix(token, "gt="))
			if ok {
				applyGtConstraint(schema, v)
			}
		case strings.HasPrefix(token, "gte="):
			v, ok := parseNumber(strings.TrimPrefix(token, "gte="))
			if ok {
				applyGteConstraint(schema, v)
			}
		case strings.HasPrefix(token, "lt="):
			v, ok := parseNumber(strings.TrimPrefix(token, "lt="))
			if ok {
				applyLtConstraint(schema, v)
			}
		case strings.HasPrefix(token, "lte="):
			v, ok := parseNumber(strings.TrimPrefix(token, "lte="))
			if ok {
				applyLteConstraint(schema, v)
			}
		}
	}
}

func parseNumber(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func applyLenConstraint(schema map[string]interface{}, v float64) {
	switch schema["type"] {
	case "string":
		schema["minLength"] = v
		schema["maxLength"] = v
	case "array":
		schema["minItems"] = v
		schema["maxItems"] = v
	}
}

func applyMinConstraint(schema map[string]interface{}, v float64) {
	switch schema["type"] {
	case "string":
		schema["minLength"] = v
	case "array":
		schema["minItems"] = v
	case "integer", "number":
		schema["minimum"] = v
	}
}

func applyMaxConstraint(schema map[string]interface{}, v float64) {
	switch schema["type"] {
	case "string":
		schema["maxLength"] = v
	case "array":
		schema["maxItems"] = v
	case "integer", "number":
		schema["maximum"] = v
	}
}

func applyGtConstraint(schema map[string]interface{}, v float64) {
	if schema["type"] == "integer" || schema["type"] == "number" {
		schema["minimum"] = v
		schema["exclusiveMinimum"] = true
	}
}

func applyGteConstraint(schema map[string]interface{}, v float64) {
	if schema["type"] == "integer" || schema["type"] == "number" {
		schema["minimum"] = v
	}
}

func applyLtConstraint(schema map[string]interface{}, v float64) {
	if schema["type"] == "integer" || schema["type"] == "number" {
		schema["maximum"] = v
		schema["exclusiveMaximum"] = true
	}
}

func applyLteConstraint(schema map[string]interface{}, v float64) {
	if schema["type"] == "integer" || schema["type"] == "number" {
		schema["maximum"] = v
	}
}

func parseEnumValues(values []string, t reflect.Type) []interface{} {
	enumValues := make([]interface{}, 0, len(values))
	for _, raw := range values {
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
				enumValues = append(enumValues, v)
				continue
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
				enumValues = append(enumValues, v)
				continue
			}
		case reflect.Float32, reflect.Float64:
			if v, err := strconv.ParseFloat(raw, 64); err == nil {
				enumValues = append(enumValues, v)
				continue
			}
		case reflect.Bool:
			if v, err := strconv.ParseBool(raw); err == nil {
				enumValues = append(enumValues, v)
				continue
			}
		}

		enumValues = append(enumValues, raw)
	}
	return enumValues
}
