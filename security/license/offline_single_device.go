package license

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNoPublicKeysConfigured indicates offline verification cannot start
	// because no embedded public keys were provided.
	ErrNoPublicKeysConfigured = errors.New("no public keys configured")
)

// NewOfflineVerifier builds a verifier for offline license checking using
// embedded package public keys and runtime hardware detection.
func NewOfflineVerifier() (*Verifier, error) {
	if len(PublicKeys) == 0 {
		return nil, ErrNoPublicKeysConfigured
	}
	return NewVerifier(
		WithPublicKeyProvider(StaticPublicKeyProvider(PublicKeys)),
	)
}

// VerifyOfflineSingleDevice verifies a token fully offline and requires
// hardware binding to match the current host.
func VerifyOfflineSingleDevice(ctx context.Context, token string, now time.Time) (*LicensePayload, error) {
	verifier, err := NewOfflineVerifier()
	if err != nil {
		return nil, err
	}
	return verifier.VerifyForCurrentHost(ctx, token, now)
}
