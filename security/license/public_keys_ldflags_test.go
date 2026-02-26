package license

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestLoadPublicKeysFromLdflags_NoValue(t *testing.T) {
	keys, ok, err := loadPublicKeysFromLdflags("")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false, got true")
	}
	if keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}
}

func TestLoadPublicKeysFromLdflags_B64(t *testing.T) {
	raw := `{"kid-a":"pub-a","kid-b":"pub-b"}`
	keys, ok, err := loadPublicKeysFromLdflags(base64.RawURLEncoding.EncodeToString([]byte(raw)))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true, got false")
	}
	if got := keys["kid-a"]; got != "pub-a" {
		t.Fatalf("expected kid-a=pub-a, got %q", got)
	}
	if got := keys["kid-b"]; got != "pub-b" {
		t.Fatalf("expected kid-b=pub-b, got %q", got)
	}
}

func TestLoadPublicKeysFromLdflags_InvalidJSON(t *testing.T) {
	raw := `{invalid`
	_, _, err := loadPublicKeysFromLdflags(base64.RawURLEncoding.EncodeToString([]byte(raw)))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse JSON") {
		t.Fatalf("expected parse JSON error, got %v", err)
	}
}

func TestLoadPublicKeysFromLdflags_EmptyValueRejected(t *testing.T) {
	raw := `{"kid-a":""}`
	_, _, err := loadPublicKeysFromLdflags(base64.RawURLEncoding.EncodeToString([]byte(raw)))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty public key") {
		t.Fatalf("expected empty public key error, got %v", err)
	}
}
