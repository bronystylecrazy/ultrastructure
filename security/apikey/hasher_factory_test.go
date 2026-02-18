package apikey

import "testing"

func TestNewHasherFromConfig_DefaultArgon2id(t *testing.T) {
	h, err := NewHasherFromConfig(Config{})
	if err != nil {
		t.Fatalf("NewHasherFromConfig: %v", err)
	}
	if _, ok := h.(*Argon2idHasher); !ok {
		t.Fatalf("expected *Argon2idHasher, got %T", h)
	}
}

func TestNewHasherFromConfig_HMAC(t *testing.T) {
	h, err := NewHasherFromConfig(Config{
		HasherMode: HasherHMACSHA256,
		HMACSecret: "top-secret",
	})
	if err != nil {
		t.Fatalf("NewHasherFromConfig: %v", err)
	}
	if _, ok := h.(*HMACSHA256Hasher); !ok {
		t.Fatalf("expected *HMACSHA256Hasher, got %T", h)
	}
}

func TestNewHasherFromConfig_HMACMissingSecret(t *testing.T) {
	_, err := NewHasherFromConfig(Config{
		HasherMode: HasherHMACSHA256,
	})
	if err == nil {
		t.Fatal("expected error for missing hmac_secret")
	}
}

func TestNewHasherFromConfig_UnsupportedMode(t *testing.T) {
	_, err := NewHasherFromConfig(Config{
		HasherMode: "unknown-mode",
	})
	if err == nil {
		t.Fatal("expected error for unsupported mode")
	}
}
