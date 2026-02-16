package web

import (
	"fmt"
	"strings"
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

		contentType := responseMeta.ContentType
		if contentType == "" {
			contentType = "application/json"
		}

		// Check for custom example first
		var example interface{}
		if customExample, hasExample := metadata.Examples[statusCode]; hasExample {
			example = customExample
		}

		// Extract schema from type
		schema := defaultSchemaForContentType(contentType)
		if responseMeta.Type != nil {
			schema = extractor.ExtractSchemaRef(responseMeta.Type)
		}

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

		response := map[string]interface{}{
			"description": description,
			"content": map[string]interface{}{
				contentType: mediaType,
			},
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
				"application/json": map[string]interface{}{
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
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": defaultErrorRef,
					},
				},
			},
		}
	}

	return responses
}

func ensureDefaultErrorResponseRef(extractor *SchemaExtractor) string {
	const schemaName = "ErrorResponse"
	if _, exists := extractor.schemas[schemaName]; !exists {
		extractor.schemas[schemaName] = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"error": map[string]interface{}{
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
	case contentType == "application/octet-stream":
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
