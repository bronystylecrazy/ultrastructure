package license

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidLicense = errors.New("invalid license")
)

type LicenseVerifier struct {
}

func NewLicenseVerifier() *LicenseVerifier {
	return &LicenseVerifier{}
}

func (v *LicenseVerifier) VerifyLicense(ctx context.Context, token string, expectedDevice *DeviceBinding, now time.Time) (*LicensePayload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("%w: empty token", ErrInvalidLicense)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	payload, signedBytes, err := parseToken(token)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRequiredFields(payload); err != nil {
		return nil, err
	}
	if err := verifySignature(payload, signedBytes); err != nil {
		return nil, err
	}
	if err := validateTime(payload, now); err != nil {
		return nil, err
	}
	if err := validateDeviceBinding(payload, expectedDevice); err != nil {
		return nil, err
	}

	return payload, nil
}

func parseToken(token string) (*LicensePayload, []byte, error) {
	raw := strings.TrimSpace(token)
	var payloadBytes []byte
	if strings.HasPrefix(raw, "{") {
		payloadBytes = []byte(raw)
	} else {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: decode token: %v", ErrInvalidLicense, err)
		}
		payloadBytes = decoded
	}

	var payload LicensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, nil, fmt.Errorf("%w: decode payload: %v", ErrInvalidLicense, err)
	}
	if strings.TrimSpace(payload.Sig) == "" {
		return nil, nil, fmt.Errorf("%w: missing signature", ErrInvalidLicense)
	}

	unsigned := payload
	unsigned.Sig = ""
	signedBytes, err := json.Marshal(unsigned)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: canonical payload: %v", ErrInvalidLicense, err)
	}

	return &payload, signedBytes, nil
}

func validateRequiredFields(payload *LicensePayload) error {
	if payload == nil {
		return fmt.Errorf("%w: missing payload", ErrInvalidLicense)
	}
	if payload.V <= 0 {
		return fmt.Errorf("%w: invalid payload version", ErrInvalidLicense)
	}
	if strings.TrimSpace(payload.LicenseID) == "" {
		return fmt.Errorf("%w: missing license_id", ErrInvalidLicense)
	}
	if strings.TrimSpace(payload.ProjectID) == "" {
		return fmt.Errorf("%w: missing project_id", ErrInvalidLicense)
	}
	if strings.TrimSpace(payload.CustomerID) == "" {
		return fmt.Errorf("%w: missing customer_id", ErrInvalidLicense)
	}
	if payload.IssuedAt <= 0 {
		return fmt.Errorf("%w: invalid issued_at", ErrInvalidLicense)
	}
	if strings.TrimSpace(payload.KID) == "" {
		return fmt.Errorf("%w: missing kid", ErrInvalidLicense)
	}
	return nil
}

func verifySignature(payload *LicensePayload, signedBytes []byte) error {
	pubEncoded, ok := PublicKeys[payload.KID]
	if !ok {
		return fmt.Errorf("%w: unknown kid", ErrInvalidLicense)
	}

	pub, err := base64.RawURLEncoding.DecodeString(pubEncoded)
	if err != nil {
		return fmt.Errorf("%w: decode public key: %v", ErrInvalidLicense, err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: invalid public key size", ErrInvalidLicense)
	}

	sig, err := base64.RawURLEncoding.DecodeString(payload.Sig)
	if err != nil {
		return fmt.Errorf("%w: decode signature: %v", ErrInvalidLicense, err)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("%w: invalid signature size", ErrInvalidLicense)
	}

	if !ed25519.Verify(ed25519.PublicKey(pub), signedBytes, sig) {
		return fmt.Errorf("%w: signature verification failed", ErrInvalidLicense)
	}
	return nil
}

func validateTime(payload *LicensePayload, now time.Time) error {
	nowUnix := now.UTC().Unix()
	if !payload.NeverExpires {
		if payload.Expiry == nil {
			return fmt.Errorf("%w: missing expiry", ErrInvalidLicense)
		}
		if nowUnix > *payload.Expiry {
			return fmt.Errorf("%w: expired", ErrInvalidLicense)
		}
	}
	return nil
}

func validateDeviceBinding(payload *LicensePayload, expected *DeviceBinding) error {
	if expected == nil {
		return nil
	}
	actual := payload.DeviceBind
	if strings.TrimSpace(actual.Platform) == "" || strings.TrimSpace(actual.Method) == "" || strings.TrimSpace(actual.PubHash) == "" {
		return fmt.Errorf("%w: invalid device binding", ErrInvalidLicense)
	}
	if expected.Platform != actual.Platform || expected.Method != actual.Method || expected.PubHash != actual.PubHash {
		return fmt.Errorf("%w: device binding mismatch", ErrInvalidLicense)
	}
	return nil
}
