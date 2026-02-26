package license

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"time"
)

var (
	ErrChallengeFailed      = errors.New("device challenge failed")
	ErrUnsupportedPublicKey = errors.New("unsupported public key type")
	ErrInvalidChallengeSig  = errors.New("invalid challenge signature")
)

// SecureEnclaveSigner represents a macOS secure-key provider.
// PublicKeyDER should return a stable DER-encoded PKIX public key.
// SignDigest signs a SHA-256 digest produced by the verifier challenge.
type SecureEnclaveSigner interface {
	PublicKeyDER(ctx context.Context) ([]byte, error)
	SignDigest(ctx context.Context, digest []byte) ([]byte, error)
}

// ExpectedHardwareBindingFromSecureEnclave derives the runtime hardware binding
// from a secure-key public key so licenses can bind to that key material.
func ExpectedHardwareBindingFromSecureEnclave(ctx context.Context, signer SecureEnclaveSigner) (*HardwareBinding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if signer == nil {
		return nil, fmt.Errorf("%w: nil signer", ErrHardwareBindingUnavailable)
	}

	pubDER, err := signer.PublicKeyDER(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: read public key: %v", ErrHardwareBindingUnavailable, err)
	}
	if len(pubDER) == 0 {
		return nil, fmt.Errorf("%w: empty public key", ErrHardwareBindingUnavailable)
	}

	return &HardwareBinding{
		Platform: "macos",
		Method:   "secure-enclave-key",
		PubHash:  hashBytesToPubHash(pubDER),
	}, nil
}

// VerifyWithSecureEnclaveChallenge verifies license payload integrity and then
// proves the process can sign a fresh challenge with the same secure key.
func (v *Verifier) VerifyWithSecureEnclaveChallenge(
	ctx context.Context,
	signer SecureEnclaveSigner,
	token string,
	now time.Time,
) (*LicensePayload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if signer == nil {
		return nil, fmt.Errorf("%w: nil signer", ErrChallengeFailed)
	}

	expected, err := ExpectedHardwareBindingFromSecureEnclave(ctx, signer)
	if err != nil {
		return nil, err
	}

	payload, err := v.Verify(ctx, token, expected, now)
	if err != nil {
		return nil, err
	}

	if err := verifySecureChallenge(ctx, signer); err != nil {
		return nil, err
	}

	return payload, nil
}

func verifySecureChallenge(ctx context.Context, signer SecureEnclaveSigner) error {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return fmt.Errorf("%w: challenge random: %v", ErrChallengeFailed, err)
	}
	digest := sha256.Sum256(challenge)

	signature, err := signer.SignDigest(ctx, digest[:])
	if err != nil {
		return fmt.Errorf("%w: sign challenge: %v", ErrChallengeFailed, err)
	}

	pubDER, err := signer.PublicKeyDER(ctx)
	if err != nil {
		return fmt.Errorf("%w: reload public key: %v", ErrChallengeFailed, err)
	}
	pubAny, err := x509.ParsePKIXPublicKey(pubDER)
	if err != nil {
		return fmt.Errorf("%w: parse public key: %v", ErrChallengeFailed, err)
	}

	ecdsaPub, ok := pubAny.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: %T", ErrUnsupportedPublicKey, pubAny)
	}
	if !ecdsa.VerifyASN1(ecdsaPub, digest[:], signature) {
		return fmt.Errorf("%w", ErrInvalidChallengeSig)
	}
	return nil
}
