package license

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

type softwareSecureSigner struct {
	priv *ecdsa.PrivateKey
	pub  []byte
	bad  bool
}

func newSoftwareSecureSigner(t *testing.T) *softwareSecureSigner {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ecdsa key: %v", err)
	}
	pub, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	return &softwareSecureSigner{priv: priv, pub: pub}
}

func (s *softwareSecureSigner) PublicKeyDER(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.pub...), nil
}

func (s *softwareSecureSigner) SignDigest(ctx context.Context, digest []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.bad {
		wrong := sha256.Sum256([]byte("wrong"))
		return ecdsa.SignASN1(rand.Reader, s.priv, wrong[:])
	}
	return ecdsa.SignASN1(rand.Reader, s.priv, digest)
}

func TestVerifyLicenseWithSecureEnclaveChallenge_OK(t *testing.T) {
	licensePub, licensePriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate license key: %v", err)
	}
	restore := swapPublicKeysForTest(map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(licensePub),
	})
	defer restore()

	signer := newSoftwareSecureSigner(t)
	pubDER, err := signer.PublicKeyDER(context.Background())
	if err != nil {
		t.Fatalf("get signer public key: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(time.Hour).Unix()
	payload := LicensePayload{
		V:          1,
		LicenseID:  "lic-se-1",
		ProjectID:  "proj-1",
		CustomerID: "cust-1",
		IssuedAt:   now.Add(-time.Minute).Unix(),
		Expiry:     &expiry,
		KID:        "kid-1",
		DeviceBind: DeviceBinding{
			Platform: "macos",
			Method:   "secure-enclave-key",
			PubHash:  hashBytesToPubHash(pubDER),
		},
		Nonce: "nonce-1",
	}
	token := mustSignToken(t, payload, licensePriv)

	verifier := NewLicenseVerifier()
	got, err := VerifyLicenseWithSecureEnclaveChallenge(context.Background(), verifier, signer, token, now)
	if err != nil {
		t.Fatalf("VerifyLicenseWithSecureEnclaveChallenge: %v", err)
	}
	if got.LicenseID != payload.LicenseID {
		t.Fatalf("expected license_id %q, got %q", payload.LicenseID, got.LicenseID)
	}
}

func TestVerifyLicenseWithSecureEnclaveChallenge_BadChallengeSig(t *testing.T) {
	licensePub, licensePriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate license key: %v", err)
	}
	restore := swapPublicKeysForTest(map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(licensePub),
	})
	defer restore()

	signer := newSoftwareSecureSigner(t)
	pubDER, err := signer.PublicKeyDER(context.Background())
	if err != nil {
		t.Fatalf("get signer public key: %v", err)
	}

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(time.Hour).Unix()
	payload := LicensePayload{
		V:          1,
		LicenseID:  "lic-se-1",
		ProjectID:  "proj-1",
		CustomerID: "cust-1",
		IssuedAt:   now.Add(-time.Minute).Unix(),
		Expiry:     &expiry,
		KID:        "kid-1",
		DeviceBind: DeviceBinding{
			Platform: "macos",
			Method:   "secure-enclave-key",
			PubHash:  hashBytesToPubHash(pubDER),
		},
		Nonce: "nonce-1",
	}
	token := mustSignToken(t, payload, licensePriv)

	signer.bad = true
	verifier := NewLicenseVerifier()
	_, err = VerifyLicenseWithSecureEnclaveChallenge(context.Background(), verifier, signer, token, now)
	if !errors.Is(err, ErrInvalidChallengeSig) {
		t.Fatalf("expected ErrInvalidChallengeSig, got: %v", err)
	}
}

func TestVerifyLicenseWithSecureEnclaveChallenge_DeviceBindingMismatch(t *testing.T) {
	licensePub, licensePriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate license key: %v", err)
	}
	restore := swapPublicKeysForTest(map[string]string{
		"kid-1": base64.RawURLEncoding.EncodeToString(licensePub),
	})
	defer restore()

	signer := newSoftwareSecureSigner(t)

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(time.Hour).Unix()
	payload := LicensePayload{
		V:          1,
		LicenseID:  "lic-se-1",
		ProjectID:  "proj-1",
		CustomerID: "cust-1",
		IssuedAt:   now.Add(-time.Minute).Unix(),
		Expiry:     &expiry,
		KID:        "kid-1",
		DeviceBind: DeviceBinding{
			Platform: "macos",
			Method:   "secure-enclave-key",
			PubHash:  "mismatch",
		},
		Nonce: "nonce-1",
	}
	token := mustSignToken(t, payload, licensePriv)

	verifier := NewLicenseVerifier()
	_, err = VerifyLicenseWithSecureEnclaveChallenge(context.Background(), verifier, signer, token, now)
	if !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected ErrInvalidLicense, got: %v", err)
	}
}

func TestExpectedDeviceBindingFromSecureEnclave_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	signer := newSoftwareSecureSigner(t)
	_, err := ExpectedDeviceBindingFromSecureEnclave(ctx, signer)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}
