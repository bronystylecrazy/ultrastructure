//go:build !darwin

package license

import (
	"context"
	"fmt"
	"runtime"
)

type MacOSKeychainSecureEnclaveSigner struct {
	keyTag          string
	createIfMissing bool
}

func NewMacOSKeychainSecureEnclaveSigner(keyTag string, createIfMissing bool) *MacOSKeychainSecureEnclaveSigner {
	return &MacOSKeychainSecureEnclaveSigner{
		keyTag:          keyTag,
		createIfMissing: createIfMissing,
	}
}

func (s *MacOSKeychainSecureEnclaveSigner) PublicKeyDER(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: secure enclave signer unsupported on %s", ErrDeviceBindingUnavailable, runtime.GOOS)
}

func (s *MacOSKeychainSecureEnclaveSigner) SignDigest(ctx context.Context, digest []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: secure enclave signer unsupported on %s", ErrDeviceBindingUnavailable, runtime.GOOS)
}
