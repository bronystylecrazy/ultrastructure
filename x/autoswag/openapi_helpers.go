package autoswag

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/bronystylecrazy/ultrastructure/web"
)

// generateSummaryFromMetadata generates a summary using metadata or falls back to auto-generation
func generateSummaryFromMetadata(route RouteInfo, metadata *RouteMetadata) string {
	if metadata != nil && metadata.Summary != "" {
		return metadata.Summary
	}
	return generateSummary(route)
}

// generateResponsesFromMetadata generates responses using registered types
func generateResponsesFromMetadata(metadata *RouteMetadata, extractor *SchemaExtractor) map[string]interface{} {
	responses := make(map[string]interface{})
	defaultErrorRef := ensureDefaultErrorResponseRef(extractor)

	for statusCode, responseMeta := range metadata.Responses {
		statusStr := fmt.Sprintf("%d", statusCode)
		description := responseMeta.Description
		if description == "" {
			description = getStatusDescription(statusCode)
		}

		// 204 and explicitly configured no-content responses should not include content.
		if responseMeta.NoContent || statusCode == 204 {
			response := map[string]interface{}{
				"description": description,
			}
			if headers := buildResponseHeaders(responseMeta.Headers); len(headers) > 0 {
				response["headers"] = headers
			}
			responses[statusStr] = response
			continue
		}

		// Check for custom example first
		var example interface{}
		if customExample, hasExample := metadata.Examples[statusCode]; hasExample {
			example = customExample
		}

		content := make(map[string]interface{})
		for contentType, modelTypes := range resolveResponseContentModels(responseMeta) {
			schema := buildResponseSchemaForModels(contentType, modelTypes, extractor)

			mediaType := map[string]interface{}{
				"schema": schema,
			}
			// For $ref schemas, put custom examples on mediaType object
			// instead of as sibling fields on the schema object.
			if example != nil {
				if _, isRef := schema["$ref"]; isRef {
					mediaType["example"] = example
				} else {
					schema["example"] = example
				}
			}
			content[contentType] = mediaType
		}

		response := map[string]interface{}{
			"description": description,
			"content":     content,
		}
		if headers := buildResponseHeaders(responseMeta.Headers); len(headers) > 0 {
			response["headers"] = headers
		}
		responses[statusStr] = response
	}

	// Add common error responses if not explicitly defined
	if _, has400 := responses["400"]; !has400 {
		responses["400"] = map[string]interface{}{
			"description": "Bad Request",
			"content": map[string]interface{}{
				web.ContentTypeApplicationJSON: map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": defaultErrorRef,
					},
				},
			},
		}
	}

	if _, has500 := responses["500"]; !has500 {
		responses["500"] = map[string]interface{}{
			"description": "Internal Server Error",
			"content": map[string]interface{}{
				web.ContentTypeApplicationJSON: map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": defaultErrorRef,
					},
				},
			},
		}
	}

	return responses
}

func resolveResponseContentModels(responseMeta ResponseMetadata) map[string][]reflect.Type {
	out := map[string][]reflect.Type{}

	for contentType, types := range responseMeta.ContentVariants {
		normalized := strings.TrimSpace(contentType)
		if normalized == "" {
			normalized = web.ContentTypeApplicationJSON
		}
		for _, t := range types {
			out[normalized] = appendUniqueModelType(out[normalized], t)
		}
	}

	for contentType, t := range responseMeta.Content {
		normalized := strings.TrimSpace(contentType)
		if normalized == "" {
			normalized = web.ContentTypeApplicationJSON
		}
		out[normalized] = appendUniqueModelType(out[normalized], t)
	}

	if len(out) > 0 {
		return out
	}

	contentType := strings.TrimSpace(responseMeta.ContentType)
	if contentType == "" {
		contentType = web.ContentTypeApplicationJSON
	}
	return map[string][]reflect.Type{
		contentType: []reflect.Type{responseMeta.Type},
	}
}

func buildResponseSchemaForModels(contentType string, modelTypes []reflect.Type, extractor *SchemaExtractor) map[string]interface{} {
	modelTypes = dedupeAndSortModelTypes(modelTypes)
	if len(modelTypes) == 0 {
		return defaultSchemaForContentType(contentType)
	}
	if len(modelTypes) == 1 {
		if modelTypes[0] == nil {
			return defaultSchemaForContentType(contentType)
		}
		return extractor.ExtractSchemaRef(modelTypes[0])
	}

	oneOf := make([]map[string]interface{}, 0, len(modelTypes))
	for _, t := range modelTypes {
		if t == nil {
			oneOf = append(oneOf, defaultSchemaForContentType(contentType))
			continue
		}
		oneOf = append(oneOf, extractor.ExtractSchemaRef(t))
	}
	return map[string]interface{}{"oneOf": oneOf}
}

func appendUniqueModelType(existing []reflect.Type, t reflect.Type) []reflect.Type {
	for _, item := range existing {
		if item == t {
			return existing
		}
	}
	return append(existing, t)
}

func dedupeAndSortModelTypes(in []reflect.Type) []reflect.Type {
	if len(in) == 0 {
		return nil
	}
	out := make([]reflect.Type, 0, len(in))
	for _, t := range in {
		out = appendUniqueModelType(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return modelTypeSortKey(out[i]) < modelTypeSortKey(out[j])
	})
	return out
}

func modelTypeSortKey(t reflect.Type) string {
	if t == nil {
		return "~nil"
	}
	return t.String()
}

func ensureDefaultErrorResponseRef(extractor *SchemaExtractor) string {
	const schemaName = "web.Error"
	if _, exists := extractor.schemas[schemaName]; !exists {
		extractor.schemas[schemaName] = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"error": map[string]interface{}{
					"type": "string",
				},
				"message": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"error"},
		}
	}
	return "#/components/schemas/" + schemaName
}

func buildResponseHeaders(headers map[string]ResponseHeaderMetadata) map[string]interface{} {
	if len(headers) == 0 {
		return nil
	}

	out := make(map[string]interface{}, len(headers))
	for name, header := range headers {
		schema := mapOpenAPIType(header.Type)
		item := map[string]interface{}{
			"schema": schema,
		}
		if header.Description != "" {
			item["description"] = header.Description
		}
		out[name] = item
	}
	return out
}

func defaultSchemaForContentType(contentType string) map[string]interface{} {
	switch {
	case strings.HasPrefix(contentType, "text/"):
		return map[string]interface{}{"type": "string"}
	case contentType == web.ContentTypeApplicationOctetStream:
		return map[string]interface{}{
			"type":   "string",
			"format": "binary",
		}
	default:
		return map[string]interface{}{"type": "object"}
	}
}

// getStatusDescription returns a human-readable description for HTTP status codes
func getStatusDescription(statusCode int) string {
	switch statusCode {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 422:
		return "Unprocessable Entity"
	case 500:
		return "Internal Server Error"
	default:
		return fmt.Sprintf("HTTP %d", statusCode)
	}
}
