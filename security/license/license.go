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

type VerificationRequest struct {
	ExpectedHardware *HardwareBinding
	Now              time.Time
}

type TokenParser interface {
	Parse(token string) (*LicensePayload, []byte, error)
}

type SignatureVerifier interface {
	Verify(ctx context.Context, payload *LicensePayload, signedBytes []byte) error
}

type PayloadValidator interface {
	Validate(ctx context.Context, payload *LicensePayload, req VerificationRequest) error
}

type PublicKeyProvider interface {
	PublicKey(ctx context.Context, kid string) (string, error)
}

type StaticPublicKeyProvider map[string]string

func (p StaticPublicKeyProvider) PublicKey(ctx context.Context, kid string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	val, ok := p[strings.TrimSpace(kid)]
	if !ok || strings.TrimSpace(val) == "" {
		return "", fmt.Errorf("%w: unknown kid", ErrInvalidLicense)
	}
	return val, nil
}

type JSONTokenParser struct{}

func (JSONTokenParser) Parse(token string) (*LicensePayload, []byte, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil, fmt.Errorf("%w: empty token", ErrInvalidLicense)
	}

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

type Ed25519SignatureVerifier struct {
	keys PublicKeyProvider
}

func NewEd25519SignatureVerifier(keys PublicKeyProvider) (*Ed25519SignatureVerifier, error) {
	if keys == nil {
		return nil, errors.New("nil public key provider")
	}
	return &Ed25519SignatureVerifier{keys: keys}, nil
}

func (v *Ed25519SignatureVerifier) Verify(ctx context.Context, payload *LicensePayload, signedBytes []byte) error {
	pubEncoded, err := v.keys.PublicKey(ctx, payload.KID)
	if err != nil {
		if errors.Is(err, ErrInvalidLicense) {
			return err
		}
		return fmt.Errorf("%w: resolve public key: %v", ErrInvalidLicense, err)
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

type RequiredFieldsValidator struct{}

func (RequiredFieldsValidator) Validate(_ context.Context, payload *LicensePayload, _ VerificationRequest) error {
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

type TimeWindowValidator struct{}

func (TimeWindowValidator) Validate(_ context.Context, payload *LicensePayload, req VerificationRequest) error {
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
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

type HardwareBindingMatchValidator struct{}

func (HardwareBindingMatchValidator) Validate(_ context.Context, payload *LicensePayload, req VerificationRequest) error {
	expected := req.ExpectedHardware
	if expected == nil {
		return nil
	}
	actual := payload.HardwareBind
	if strings.TrimSpace(actual.Platform) == "" || strings.TrimSpace(actual.Method) == "" || strings.TrimSpace(actual.PubHash) == "" {
		return fmt.Errorf("%w: invalid hardware binding", ErrInvalidLicense)
	}
	if expected.Platform != actual.Platform || expected.Method != actual.Method || expected.PubHash != actual.PubHash {
		return fmt.Errorf("%w: hardware binding mismatch", ErrInvalidLicense)
	}
	return nil
}

type Verifier struct {
	detector          HardwareDetector
	parser            TokenParser
	requiredValidator PayloadValidator
	signature         SignatureVerifier
	policyValidators  []PayloadValidator
}

type VerifierOption func(*verifierConfig) error

type verifierConfig struct {
	detector          HardwareDetector
	parser            TokenParser
	requiredValidator PayloadValidator
	signature         SignatureVerifier
	publicKeyProvider PublicKeyProvider
	policyValidators  []PayloadValidator
	explicitSignature bool
	explicitProvider  bool
}

func WithHardwareDetector(detector HardwareDetector) VerifierOption {
	return func(cfg *verifierConfig) error {
		if detector == nil {
			return errors.New("nil hardware detector")
		}
		cfg.detector = detector
		return nil
	}
}

func WithTokenParser(parser TokenParser) VerifierOption {
	return func(cfg *verifierConfig) error {
		if parser == nil {
			return errors.New("nil token parser")
		}
		cfg.parser = parser
		return nil
	}
}

func WithRequiredValidator(validator PayloadValidator) VerifierOption {
	return func(cfg *verifierConfig) error {
		if validator == nil {
			return errors.New("nil required-field validator")
		}
		cfg.requiredValidator = validator
		return nil
	}
}

func WithSignatureVerifier(signature SignatureVerifier) VerifierOption {
	return func(cfg *verifierConfig) error {
		if signature == nil {
			return errors.New("nil signature verifier")
		}
		cfg.signature = signature
		cfg.explicitSignature = true
		return nil
	}
}

func WithPublicKeyProvider(provider PublicKeyProvider) VerifierOption {
	return func(cfg *verifierConfig) error {
		if provider == nil {
			return errors.New("nil public key provider")
		}
		cfg.publicKeyProvider = provider
		cfg.explicitProvider = true
		return nil
	}
}

func WithPolicyValidators(validators ...PayloadValidator) VerifierOption {
	return func(cfg *verifierConfig) error {
		for i := range validators {
			if validators[i] == nil {
				return fmt.Errorf("nil policy validator at index %d", i)
			}
		}
		cfg.policyValidators = validators
		return nil
	}
}

func NewVerifier(opts ...VerifierOption) (*Verifier, error) {
	cfg := verifierConfig{
		detector:          NewHardwareDetector(),
		parser:            JSONTokenParser{},
		requiredValidator: RequiredFieldsValidator{},
		publicKeyProvider: StaticPublicKeyProvider(PublicKeys),
		policyValidators: []PayloadValidator{
			TimeWindowValidator{},
			HardwareBindingMatchValidator{},
		},
	}

	for i := range opts {
		if opts[i] == nil {
			return nil, fmt.Errorf("nil verifier option at index %d", i)
		}
		if err := opts[i](&cfg); err != nil {
			return nil, err
		}
	}
	if cfg.explicitSignature && cfg.explicitProvider {
		return nil, errors.New("conflicting options: WithSignatureVerifier and WithPublicKeyProvider are mutually exclusive")
	}

	if cfg.detector == nil {
		return nil, errors.New("nil hardware detector")
	}
	if cfg.parser == nil {
		return nil, errors.New("nil token parser")
	}
	if cfg.requiredValidator == nil {
		return nil, errors.New("nil required-field validator")
	}
	if cfg.signature == nil {
		signature, err := NewEd25519SignatureVerifier(cfg.publicKeyProvider)
		if err != nil {
			return nil, err
		}
		cfg.signature = signature
	}
	for i := range cfg.policyValidators {
		if cfg.policyValidators[i] == nil {
			return nil, fmt.Errorf("nil policy validator at index %d", i)
		}
	}
	return &Verifier{
		detector:          cfg.detector,
		parser:            cfg.parser,
		requiredValidator: cfg.requiredValidator,
		signature:         cfg.signature,
		policyValidators:  cfg.policyValidators,
	}, nil
}

func (v *Verifier) DetectHardwareBinding(ctx context.Context) (*HardwareBinding, error) {
	binding, err := v.detector.Detect(ctx)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, fmt.Errorf("%w: empty binding", ErrHardwareBindingUnavailable)
	}
	return binding, nil
}

func (v *Verifier) Verify(ctx context.Context, token string, expectedHardware *HardwareBinding, now time.Time) (*LicensePayload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req := VerificationRequest{
		ExpectedHardware: expectedHardware,
		Now:              now,
	}

	payload, signedBytes, err := v.parser.Parse(token)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := v.requiredValidator.Validate(ctx, payload, req); err != nil {
		return nil, err
	}
	if err := v.signature.Verify(ctx, payload, signedBytes); err != nil {
		return nil, err
	}
	for i := range v.policyValidators {
		if err := v.policyValidators[i].Validate(ctx, payload, req); err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func (v *Verifier) VerifyForCurrentHost(ctx context.Context, token string, now time.Time) (*LicensePayload, error) {
	expected, err := v.DetectHardwareBinding(ctx)
	if err != nil {
		return nil, err
	}
	return v.Verify(ctx, token, expected, now)
}
