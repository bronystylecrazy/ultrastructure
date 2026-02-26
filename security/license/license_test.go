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

	verifier := mustNewTestVerifier(t, map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	device := HardwareBinding{
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
		HardwareBind: device,
		X:            map[string]any{"max_cameras": float64(2)},
		Nonce:        "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	got, err := verifier.Verify(context.Background(), token, &device, now)
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

	verifier := mustNewTestVerifier(t, map[string]string{})

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	payload := LicensePayload{
		V:            1,
		LicenseID:    "lic-1",
		ProjectID:    "proj-1",
		CustomerID:   "cust-1",
		IssuedAt:     now.Add(-time.Minute).Unix(),
		Expiry:       &expiry,
		KID:          "kid-missing",
		HardwareBind: HardwareBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash"},
		Nonce:        "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	_, err = verifier.Verify(context.Background(), token, nil, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_InvalidSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	verifier := mustNewTestVerifier(t, map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	payload := LicensePayload{
		V:            1,
		LicenseID:    "lic-1",
		ProjectID:    "proj-1",
		CustomerID:   "cust-1",
		IssuedAt:     now.Add(-time.Minute).Unix(),
		Expiry:       &expiry,
		KID:          "kid-1",
		HardwareBind: HardwareBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash"},
		Nonce:        "nonce-1",
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

	_, err = verifier.Verify(context.Background(), string(tampered), nil, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_Expired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	verifier := mustNewTestVerifier(t, map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(-time.Minute).Unix()
	payload := LicensePayload{
		V:            1,
		LicenseID:    "lic-1",
		ProjectID:    "proj-1",
		CustomerID:   "cust-1",
		IssuedAt:     now.Add(-2 * time.Minute).Unix(),
		Expiry:       &expiry,
		KID:          "kid-1",
		HardwareBind: HardwareBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash"},
		Nonce:        "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	_, err = verifier.Verify(context.Background(), token, nil, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_DeviceMismatch(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	verifier := mustNewTestVerifier(t, map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(pub),
	})

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(30 * time.Minute).Unix()
	payload := LicensePayload{
		V:            1,
		LicenseID:    "lic-1",
		ProjectID:    "proj-1",
		CustomerID:   "cust-1",
		IssuedAt:     now.Add(-2 * time.Minute).Unix(),
		Expiry:       &expiry,
		KID:          "kid-1",
		HardwareBind: HardwareBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash-a"},
		Nonce:        "nonce-1",
	}
	token := mustSignToken(t, payload, priv)

	expected := HardwareBinding{Platform: "linux", Method: "tpm-ek-hash", PubHash: "hash-b"}
	_, err = verifier.Verify(context.Background(), token, &expected, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestVerifyLicense_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	verifier := mustNewTestVerifier(t, map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize)),
	})
	_, err := verifier.Verify(ctx, "{}", nil, time.Time{})
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

func mustNewTestVerifier(t *testing.T, keys map[string]string) *Verifier {
	t.Helper()
	v, err := NewVerifier(
		WithPublicKeyProvider(StaticPublicKeyProvider(keys)),
		WithHardwareDetector(NewHardwareDetector()),
	)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	return v
}

func TestNewVerifier_NilOption(t *testing.T) {
	_, err := NewVerifier(nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNewVerifier_NilPublicKeyProviderOption(t *testing.T) {
	_, err := NewVerifier(WithPublicKeyProvider(nil))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNewVerifier_NilPolicyValidatorOption(t *testing.T) {
	_, err := NewVerifier(WithPolicyValidators(TimeWindowValidator{}, nil))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNewVerifier_ConflictingSignatureAndProvider(t *testing.T) {
	signature, err := NewEd25519SignatureVerifier(StaticPublicKeyProvider(map[string]string{
		"kid-a": "pub-a",
	}))
	if err != nil {
		t.Fatalf("new signature verifier: %v", err)
	}
	_, err = NewVerifier(
		WithPublicKeyProvider(StaticPublicKeyProvider(map[string]string{"kid-b": "pub-b"})),
		WithSignatureVerifier(signature),
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
