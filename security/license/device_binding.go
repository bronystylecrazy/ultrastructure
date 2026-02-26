package license

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"runtime"
	"strings"
)

var (
	ErrHardwareBindingUnavailable = errors.New("hardware binding unavailable")
)

type HardwareDetector interface {
	Detect(ctx context.Context) (*HardwareBinding, error)
}

func NewHardwareDetector() HardwareDetector {
	return platformHardwareDetector{}
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

func newOSKeystoreBinding(rawID string) *HardwareBinding {
	return &HardwareBinding{
		Platform: normalizedPlatform(),
		Method:   "os-keystore",
		PubHash:  hashToPubHash(strings.ToLower(strings.TrimSpace(rawID))),
	}
}
