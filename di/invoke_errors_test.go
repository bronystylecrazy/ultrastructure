package di

import (
	"strings"
	"testing"
)

func TestInvokeRejectsParamsCountMismatch(t *testing.T) {
	err := startAppError(t,
		Supply(&basicThing{value: "a"}, Name("one")),
		Invoke(func(a *basicThing, b *basicThing) {
		}, Params(Name("one"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "invoke_errors_test.go:") {
		t.Fatalf("expected source location in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Params count must match invoke function parameter count") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "di wiring:") {
		t.Fatalf("expected wiring location marker, got: %v", err)
	}
}

func TestInvokeRejectsVariadicParamsCountMismatch(t *testing.T) {
	err := startAppError(t,
		Supply(&basicThing{value: "a"}, Name("one")),
		Invoke(func(a *basicThing, rest ...*basicThing) {
		}, Params(Name("one"))),
	)
	if err == nil {
		t.Fatal("expected start to fail")
	}
	if !strings.Contains(err.Error(), "invoke_errors_test.go:") {
		t.Fatalf("expected source location in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "expected 2, got 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
