package otel

import (
	"context"
	"testing"

	"github.com/bronystylecrazy/ultrastructure/meta"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func setVersion(t *testing.T, version string) {
	t.Helper()
	previous := meta.Version
	meta.Version = version
	t.Cleanup(func() { meta.Version = previous })
}

func serviceVersion(t *testing.T, config Config) (string, bool) {
	t.Helper()
	res, err := NewResource(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	value, ok := res.Set().Value(semconv.ServiceVersionKey)
	return value.AsString(), ok
}

func TestNewResourceCarriesTheBuildVersion(t *testing.T) {
	setVersion(t, "v1.2.3")

	got, ok := serviceVersion(t, Config{ServiceName: "svc"})
	if !ok || got != "v1.2.3" {
		t.Fatalf("service.version = %q (present %v), want %q", got, ok, "v1.2.3")
	}
}

// A development build has no version worth reporting.
func TestNewResourceOmitsTheVersionOfADevelopmentBuild(t *testing.T) {
	setVersion(t, meta.NilVersion)

	if got, ok := serviceVersion(t, Config{ServiceName: "svc"}); ok {
		t.Fatalf("service.version = %q, want it absent", got)
	}
}

func TestNewResourceLetsAnExplicitVersionWin(t *testing.T) {
	setVersion(t, "v1.2.3")

	got, _ := serviceVersion(t, Config{
		ServiceName:   "svc",
		ResourceAttrs: map[string]string{"service.version": "2026.11-hotfix"},
	})
	if got != "2026.11-hotfix" {
		t.Fatalf("service.version = %q, want %q", got, "2026.11-hotfix")
	}
}
