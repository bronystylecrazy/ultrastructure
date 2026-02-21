package analyzer

import (
	"strings"
	"testing"
)

func TestGenerateHookSource_IncludesDetectedMetadata(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/sample"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	mustContain := []string{
		"package autodoc",
		"func GeneratedHook(ctx *autoswag.Context)",
		"switch routeKey {",
		"case \"POST /templates/{template_id}/bulk\":",
		"ctx.SetRequestBody(sample.CreateBulkRequest{}, true, fiber.MIMEApplicationJSON)",
		"ctx.AddParameter(autoswag.ParameterMetadata{Name: \"template_id\", In: \"path\", Type: reflect.TypeOf(int64(0)), Required: true})",
		"ctx.SetResponseAs(http.StatusBadRequest, web.Error{}, fiber.MIMEApplicationJSON, \"Auto-detected\")",
		"ctx.SetResponseAs(http.StatusCreated, web.Response{}, fiber.MIMEApplicationJSON, \"Auto-detected\")",
	}
	for _, want := range mustContain {
		if !strings.Contains(src, want) {
			t.Fatalf("expected generated source to contain %q\nsource:\n%s", want, src)
		}
	}
}

func TestGenerateHookSource_DefaultFuncNameIsAutoSwagGenerator(t *testing.T) {
	report := &Report{}
	src, err := GenerateHookSource(report, GenerateOptions{})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}
	if !strings.Contains(src, "func AutoSwagGenerator(ctx *autoswag.Context)") {
		t.Fatalf("expected default generated function name AutoSwagGenerator, source:\n%s", src)
	}
}

func TestGenerateHookSource_ImportAliasCollision(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path:    "example/pkg1",
				Imports: map[string]string{"model": "example.com/a/model"},
				Handlers: []HandlerReport{
					{
						Name: "H1",
						Key:  "k1",
						Responses: []ResponseTypeReport{
							{Status: 200, Type: "model.User", ContentType: "application/json"},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/a", HandlerKey: "k1"},
				},
			},
			{
				Path:    "example/pkg2",
				Imports: map[string]string{"model": "example.com/b/model"},
				Handlers: []HandlerReport{
					{
						Name: "H2",
						Key:  "k2",
						Responses: []ResponseTypeReport{
							{Status: 200, Type: "model.User", ContentType: "application/json"},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/b", HandlerKey: "k2"},
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	mustContain := []string{
		"model \"example.com/a/model\"",
		"model2 \"example.com/b/model\"",
		"ctx.SetResponseAs(http.StatusOK, model.User{}, fiber.MIMEApplicationJSON, \"Auto-detected\")",
		"ctx.SetResponseAs(http.StatusOK, model2.User{}, fiber.MIMEApplicationJSON, \"Auto-detected\")",
	}
	for _, want := range mustContain {
		if !strings.Contains(src, want) {
			t.Fatalf("expected generated source to contain %q\nsource:\n%s", want, src)
		}
	}
}

func TestGenerateHookSource_ExactOnlySkipsNonExactResponses(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path:    "example/pkg",
				Imports: map[string]string{"web": "github.com/bronystylecrazy/ultrastructure/web"},
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
						Responses: []ResponseTypeReport{
							{Status: 200, Type: "web.Response", ContentType: "application/json", Confidence: "exact"},
							{Status: 500, Type: "web.Error", ContentType: "application/json", Confidence: "inferred"},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/x", HandlerKey: "k"},
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
		ExactOnly:   true,
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	if !strings.Contains(src, "ctx.SetResponseAs(http.StatusOK, web.Response{}, fiber.MIMEApplicationJSON, \"Auto-detected\")") {
		t.Fatalf("expected exact response to be emitted, source:\n%s", src)
	}
	if strings.Contains(src, "ctx.SetResponseAs(http.StatusInternalServerError, web.Error{}, fiber.MIMEApplicationJSON, \"Auto-detected\")") {
		t.Fatalf("expected inferred response to be skipped in exact-only mode, source:\n%s", src)
	}
}

func TestGenerateHookSource_ExactOnlySkipsInferredRequestAndPath(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path:    "example/pkg",
				Imports: map[string]string{"web": "github.com/bronystylecrazy/ultrastructure/web"},
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
						Request: &RequestReport{
							Type:         "web.Response",
							ContentTypes: []string{"application/json"},
							Confidence:   "inferred",
						},
						Path: []PathParamReport{
							{Name: "slug", Type: "string", Confidence: "inferred"},
							{Name: "id", Type: "integer", Confidence: "exact"},
						},
						Responses: []ResponseTypeReport{
							{Status: 200, Type: "web.Response", ContentType: "application/json", Confidence: "exact"},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/x/{id}/{slug}", HandlerKey: "k"},
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
		ExactOnly:   true,
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	if strings.Contains(src, "ctx.SetRequestBody(") {
		t.Fatalf("expected inferred request body to be skipped in exact-only mode, source:\n%s", src)
	}
	if strings.Contains(src, "Name: \"slug\"") {
		t.Fatalf("expected inferred slug path param to be skipped in exact-only mode, source:\n%s", src)
	}
	if !strings.Contains(src, "Name: \"id\"") {
		t.Fatalf("expected exact id path param to be emitted in exact-only mode, source:\n%s", src)
	}
}

func TestGenerateHookSource_EmitsQueryMetadata(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path:    "example/pkg",
				Imports: map[string]string{"web": "github.com/bronystylecrazy/ultrastructure/web"},
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
						Query: &TypeReport{
							Type:       "web.Error",
							Confidence: "inferred",
						},
						Responses: []ResponseTypeReport{
							{Status: 200, Type: "web.Response", ContentType: "application/json", Confidence: "exact"},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/x", HandlerKey: "k"},
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	if !strings.Contains(src, "ctx.SetQuery(web.Error{})") {
		t.Fatalf("expected query metadata to be emitted, source:\n%s", src)
	}
}

func TestGenerateHookSource_EmitsRouteNameAndDescription(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path: "example/pkg",
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
					},
				},
				Routes: []RouteBindingReport{
					{
						Method:      "GET",
						Path:        "/x",
						HandlerKey:  "k",
						Name:        "ListX",
						Description: "Lists x resources",
						Tags:        []string{"users", "v1"},
						TagDescriptions: map[string]string{
							"users": "Users API",
						},
						Responses: []ResponseTypeReport{
							{
								Status:      201,
								Type:        "string",
								ContentType: "text/plain",
								Description: "Created text",
							},
						},
						PathParams: []PathParamReport{
							{
								Name:        "id",
								Type:        "string",
								Description: "Resource ID",
							},
						},
						ResponseHeaders: []RouteResponseHeaderReport{
							{Status: 201, Name: "X-Request-ID", Type: "string", Description: "Correlation id"},
						},
					},
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}
	if !strings.Contains(src, "ctx.SetSummary(\"ListX\")") {
		t.Fatalf("expected route summary emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.SetDescription(\"Lists x resources\")") {
		t.Fatalf("expected route description emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.AddTag(\"users\", \"v1\")") {
		t.Fatalf("expected route tag emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.AddTagDescription(\"users\", \"Users API\")") {
		t.Fatalf("expected route tag description emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.SetResponseAs(http.StatusCreated, \"\", fiber.MIMETextPlain, \"Created text\")") {
		t.Fatalf("expected route response override emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.AddResponseHeader(http.StatusCreated, \"X-Request-ID\", reflect.TypeOf(\"\"), \"Correlation id\")") {
		t.Fatalf("expected route response header override emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.AddParameter(autoswag.ParameterMetadata{Name: \"id\", In: \"path\", Type: reflect.TypeOf(\"\"), Required: true, Description: \"Resource ID\"})") {
		t.Fatalf("expected route path param override emission, source:\n%s", src)
	}
}

func TestGenerateHookSource_UsesDetectedResponseDescription(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path: "example/pkg",
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
						Responses: []ResponseTypeReport{
							{
								Status:      200,
								Type:        "string",
								ContentType: "text/plain",
								Description: "This is example command",
							},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/x", HandlerKey: "k"},
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	if !strings.Contains(src, "ctx.SetResponseAs(http.StatusOK, \"\", fiber.MIMETextPlain, \"This is example command\")") {
		t.Fatalf("expected generated source to use detected response description, source:\n%s", src)
	}
}

func TestGenerateHookSource_MergesDuplicateRouteBodies(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path: "example/pkg",
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
						Responses: []ResponseTypeReport{
							{
								Status:      200,
								Type:        "string",
								ContentType: "text/plain",
								Description: "Auto-detected",
							},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/healthz", HandlerKey: "k"},
					{Method: "GET", Path: "/readyz", HandlerKey: "k"},
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	if !strings.Contains(src, "case \"GET /healthz\", \"GET /readyz\":") {
		t.Fatalf("expected merged case labels for identical bodies, source:\n%s", src)
	}
}

func TestGenerateHookSource_EmitsResponseHeaders(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path: "example/pkg",
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
						Responses: []ResponseTypeReport{
							{
								Status:      201,
								Type:        "web.Response",
								ContentType: "application/json",
								Headers: map[string]ResponseHeaderReport{
									"X-Request-ID": {Type: "string", Description: "Auto-detected"},
									"X-Retry-After": {Type: "integer", Description: "Retry delay"},
								},
							},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "POST", Path: "/x", HandlerKey: "k"},
				},
				Imports: map[string]string{
					"web": "github.com/bronystylecrazy/ultrastructure/web",
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	if !strings.Contains(src, "ctx.AddResponseHeader(http.StatusCreated, \"X-Request-ID\", reflect.TypeOf(\"\"), \"Auto-detected\")") {
		t.Fatalf("expected AddResponseHeader string emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.AddResponseHeader(http.StatusCreated, \"X-Retry-After\", reflect.TypeOf(int64(0)), \"Retry delay\")") {
		t.Fatalf("expected AddResponseHeader integer emission, source:\n%s", src)
	}
}

func TestGenerateHookSource_EmitsModelFieldDescriptions(t *testing.T) {
	report := &Report{
		Packages: []PackageReport{
			{
				Path: "example/pkg",
				Handlers: []HandlerReport{
					{
						Name: "H",
						Key:  "k",
						Request: &RequestReport{
							Type:         "web.Response",
							ContentTypes: []string{"application/json"},
							Fields: map[string]string{
								"Message": "payload message",
							},
						},
						Query: &TypeReport{
							Type: "web.Error",
							Fields: map[string]string{
								"Error": "query error",
							},
						},
						Responses: []ResponseTypeReport{
							{
								Status:      200,
								Type:        "web.Response",
								ContentType: "application/json",
								Fields: map[string]string{
									"Message": "response message",
								},
							},
						},
					},
				},
				Routes: []RouteBindingReport{
					{Method: "GET", Path: "/x", HandlerKey: "k"},
				},
				Imports: map[string]string{
					"web": "github.com/bronystylecrazy/ultrastructure/web",
				},
			},
		},
	}

	src, err := GenerateHookSource(report, GenerateOptions{
		PackageName: "autodoc",
		FuncName:    "GeneratedHook",
	})
	if err != nil {
		t.Fatalf("GenerateHookSource returned error: %v", err)
	}

	if !strings.Contains(src, "ctx.AddModelFieldDescription(web.Response{}, \"Message\", \"payload message\")") {
		t.Fatalf("expected request model field description emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.AddModelFieldDescription(web.Error{}, \"Error\", \"query error\")") {
		t.Fatalf("expected query model field description emission, source:\n%s", src)
	}
	if !strings.Contains(src, "ctx.AddModelFieldDescription(web.Response{}, \"Message\", \"response message\")") {
		t.Fatalf("expected response model field description emission, source:\n%s", src)
	}
}
