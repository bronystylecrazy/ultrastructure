package license

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestVerifyLicense_OK(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	restore := swapPublicKeysForTest(map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})
	defer restore()

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	device := DeviceBinding{
		Platform: "linux",
		Method:   "tpm-ek-hash",
		PubHash:  "hash",
	}

	payload := LicensePayload{
		V:            1,
		LicenseID:    "lic-1",
		ProjectID:    "proj-1",
		CustomerID:   "cust-1",
		IssuedAt:     now.Add(-time.Minute).Unix(),
		Expiry:       &expiry,
		NeverExpires: false,
		KID:          "kid-1",
		DeviceBind:   device,
		X:            map[string]any{"max_cameras": float64(2)},
		Nonce:        "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	v := NewLicenseVerifier()
	got, err := v.VerifyLicense(context.Background(), token, &device, now)
	if err != nil {
		t.Fatalf("VerifyLicense: %v", err)
	}
	if got.LicenseID != payload.LicenseID {
		t.Fatalf("expected license_id %q, got %q", payload.LicenseID, got.LicenseID)
	}
	if got.KID != payload.KID {
		t.Fatalf("expected kid %q, got %q", payload.KID, got.KID)
	}
}

func TestVerifyLicense_UnknownKID(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	restore := swapPublicKeysForTest(map[string]string{})
	defer restore()

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	payload := LicensePayload{
		V:          1,
		LicenseID:  "lic-1",
		ProjectID:  "proj-1",
		CustomerID: "cust-1",
		IssuedAt:   now.Add(-time.Minute).Unix(),
		Expiry:     &expiry,
		KID:        "kid-missing",
		DeviceBind: DeviceBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash"},
		Nonce:      "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	v := NewLicenseVerifier()
	_, err = v.VerifyLicense(context.Background(), token, nil, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_InvalidSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	restore := swapPublicKeysForTest(map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})
	defer restore()

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	payload := LicensePayload{
		V:          1,
		LicenseID:  "lic-1",
		ProjectID:  "proj-1",
		CustomerID: "cust-1",
		IssuedAt:   now.Add(-time.Minute).Unix(),
		Expiry:     &expiry,
		KID:        "kid-1",
		DeviceBind: DeviceBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash"},
		Nonce:      "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	var decoded LicensePayload
	if err := json.Unmarshal([]byte(token), &decoded); err != nil {
		t.Fatalf("unmarshal token: %v", err)
	}
	decoded.ProjectID = "tampered"
	tampered, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}

	v := NewLicenseVerifier()
	_, err = v.VerifyLicense(context.Background(), string(tampered), nil, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_Expired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	restore := swapPublicKeysForTest(map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})
	defer restore()

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(-time.Minute).Unix()
	payload := LicensePayload{
		V:          1,
		LicenseID:  "lic-1",
		ProjectID:  "proj-1",
		CustomerID: "cust-1",
		IssuedAt:   now.Add(-2 * time.Minute).Unix(),
		Expiry:     &expiry,
		KID:        "kid-1",
		DeviceBind: DeviceBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash"},
		Nonce:      "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	v := NewLicenseVerifier()
	_, err = v.VerifyLicense(context.Background(), token, nil, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_DeviceMismatch(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	restore := swapPublicKeysForTest(map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})
	defer restore()

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	payload := LicensePayload{
		V:          1,
		LicenseID:  "lic-1",
		ProjectID:  "proj-1",
		CustomerID: "cust-1",
		IssuedAt:   now.Add(-2 * time.Minute).Unix(),
		Expiry:     &expiry,
		KID:        "kid-1",
		DeviceBind: DeviceBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash-a"},
		Nonce:      "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	expected := DeviceBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash-b"}
	v := NewLicenseVerifier()
	_, err = v.VerifyLicense(context.Background(), token, &expected, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	v := NewLicenseVerifier()
	_, err := v.VerifyLicense(ctx, "{}", nil, time.Time{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func mustSignToken(t *testing.T, payload LicensePayload, priv ed25519.PrivateKey) string {
	t.Helper()

	unsigned := payload
	unsigned.Sig = ""

	rawUnsigned, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf("marshal unsigned payload: %v", err)
	}

	payload.Sig = base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, rawUnsigned))

	rawSigned, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal signed payload: %v", err)
	}
	return string(rawSigned)
}

func swapPublicKeysForTest(next map[string]string) func() {
	prev := PublicKeys
	PublicKeys = next
	return func() {
		PublicKeys = prev
	}
}
