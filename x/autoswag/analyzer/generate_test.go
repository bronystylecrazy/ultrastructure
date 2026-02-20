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
		"ctx.SetRequestBody(sample.CreateBulkRequest{}, true, \"application/json\")",
		"ctx.AddParameter(autoswag.ParameterMetadata{Name: \"template_id\", In: \"path\", Type: reflect.TypeOf(int64(0)), Required: true})",
		"ctx.SetResponseAs(400, web.Error{}, \"application/json\", \"Auto-detected\")",
		"ctx.SetResponseAs(201, web.Response{}, \"application/json\", \"Auto-detected\")",
	}
	for _, want := range mustContain {
		if !strings.Contains(src, want) {
			t.Fatalf("expected generated source to contain %q\nsource:\n%s", want, src)
		}
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
		"ctx.SetResponseAs(200, model.User{}, \"application/json\", \"Auto-detected\")",
		"ctx.SetResponseAs(200, model2.User{}, \"application/json\", \"Auto-detected\")",
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

	if !strings.Contains(src, "ctx.SetResponseAs(200, web.Response{}, \"application/json\", \"Auto-detected\")") {
		t.Fatalf("expected exact response to be emitted, source:\n%s", src)
	}
	if strings.Contains(src, "ctx.SetResponseAs(500, web.Error{}, \"application/json\", \"Auto-detected\")") {
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

	if !strings.Contains(src, "ctx.SetResponseAs(200, \"\", \"text/plain\", \"This is example command\")") {
		t.Fatalf("expected generated source to use detected response description, source:\n%s", src)
	}
}
