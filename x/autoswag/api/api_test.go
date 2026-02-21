package api

import (
	"strings"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer"
)

func TestRun_JSONPayload(t *testing.T) {
	res, err := Run(Request{
		AnalyzerOptions: analyzer.Options{
			Dir:      ".",
			Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/sample"},
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res == nil || res.Report == nil {
		t.Fatalf("expected non-nil result/report")
	}
	if len(res.Payload) == 0 {
		t.Fatalf("expected JSON payload")
	}
	if !strings.Contains(string(res.Payload), "\"packages\"") {
		t.Fatalf("expected JSON payload to contain packages key, got %s", string(res.Payload))
	}
}

func TestRun_HookPayload(t *testing.T) {
	res, err := Run(Request{
		AnalyzerOptions: analyzer.Options{
			Dir:      ".",
			Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/sample"},
		},
		EmitHook:    true,
		HookPackage: "autodoc",
		HookName:    "GeneratedHook",
		ToolVersion: "v1.2.3",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}
	out := string(res.Payload)
	if !strings.Contains(out, "package autodoc") {
		t.Fatalf("expected generated hook package in payload, got %s", out)
	}
	if !strings.Contains(out, "func GeneratedHook(ctx *autoswag.Context)") {
		t.Fatalf("expected generated hook function in payload, got %s", out)
	}
}
