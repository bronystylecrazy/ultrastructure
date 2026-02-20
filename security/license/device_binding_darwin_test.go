//go:build darwin

package license

import "testing"

func TestParseIORegPlatformUUID(t *testing.T) {
	raw := `
{
  "IOPlatformUUID" = "A1B2C3D4-E5F6-1122-3344-556677889900"
}
`
	got, ok := parseIORegPlatformUUID(raw)
	if !ok {
		t.Fatal("expected parse success")
	}
	if got != "A1B2C3D4-E5F6-1122-3344-556677889900" {
		t.Fatalf("unexpected uuid: %q", got)
	}
}

func TestParseIORegPlatformUUID_NotFound(t *testing.T) {
	if got, ok := parseIORegPlatformUUID(`{}`); ok || got != "" {
		t.Fatalf("expected no match, got ok=%v val=%q", ok, got)
	}
}
