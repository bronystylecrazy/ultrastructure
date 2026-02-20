package license

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var (
	ErrDeviceBindingUnavailable = errors.New("device binding unavailable")
)

// ExpectedDeviceBinding returns the current machine binding in the same shape
// as the license payload so callers can compare signed and runtime values.
func ExpectedDeviceBinding(ctx context.Context) (*DeviceBinding, error) {
	binding, err := expectedDeviceBinding(ctx)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, fmt.Errorf("%w: empty binding", ErrDeviceBindingUnavailable)
	}
	return binding, nil
}

func normalizedPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

func hashToPubHash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func hashBytesToPubHash(value []byte) string {
	sum := sha256.Sum256(value)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
