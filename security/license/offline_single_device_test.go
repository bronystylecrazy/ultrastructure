package license

import (
	"errors"
	"testing"
)

func TestNewOfflineVerifier_NoKeys(t *testing.T) {
	orig := PublicKeys
	PublicKeys = map[string]string{}
	t.Cleanup(func() { PublicKeys = orig })

	_, err := NewOfflineVerifier()
	if !errors.Is(err, ErrNoPublicKeysConfigured) {
		t.Fatalf("expected ErrNoPublicKeysConfigured, got: %v", err)
	}
}

func TestNewOfflineVerifier_WithKeys(t *testing.T) {
	orig := PublicKeys
	PublicKeys = map[string]string{
		"kid-1": "pub-1",
	}
	t.Cleanup(func() { PublicKeys = orig })

	verifier, err := NewOfflineVerifier()
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if verifier == nil {
		t.Fatalf("expected verifier, got nil")
	}
}
