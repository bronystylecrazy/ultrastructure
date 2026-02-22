package jws

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type SignerVerifier interface {
	Sign(claims map[string]any) (string, error)
	Verify(tokenValue string) (Claims, error)
}

type Signer struct {
	config        Config
	signingAlg    string
	signingMethod jwtgo.SigningMethod
	signingKey    any
	verifyKey     any
	now           func() time.Time
}

var _ SignerVerifier = (*Signer)(nil)

func NewSigner(config Config) (*Signer, error) {
	cfg := config.withDefaults()
	signingMethod, signingKey, verifyKey, err := newSigningConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Signer{
		config:        cfg,
		signingAlg:    cfg.Algorithm,
		signingMethod: signingMethod,
		signingKey:    signingKey,
		verifyKey:     verifyKey,
		now:           time.Now,
	}, nil
}

func (s *Signer) Sign(claims map[string]any) (string, error) {
	now := s.now().UTC().Unix()
	out := jwtgo.MapClaims{}
	for k, v := range claims {
		out[k] = v
	}
	if _, ok := out["jti"]; !ok {
		out["jti"] = uuid.NewString()
	}
	if _, ok := out["iat"]; !ok {
		out["iat"] = now
	}
	if _, ok := out["nbf"]; !ok {
		out["nbf"] = now
	}
	if s.config.Issuer != "" {
		if _, ok := out["iss"]; !ok {
			out["iss"] = s.config.Issuer
		}
	}

	t := jwtgo.NewWithClaims(s.signingMethod, out)
	return t.SignedString(s.signingKey)
}

func (s *Signer) Verify(tokenValue string) (Claims, error) {
	token, err := jwtgo.Parse(tokenValue, func(token *jwtgo.Token) (any, error) {
		gotAlg := ""
		if token != nil && token.Method != nil {
			gotAlg = token.Method.Alg()
		}
		if gotAlg != s.signingAlg {
			return nil, fmt.Errorf("%w: got=%s want=%s", ErrUnexpectedTokenAlg, gotAlg, s.signingAlg)
		}
		return s.verifyKey, nil
	})
	if err != nil {
		return Claims{}, err
	}
	if !token.Valid {
		return Claims{}, jwtgo.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwtgo.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidClaims
	}
	return claimsFromJWT(claims), nil
}

func newSigningConfig(cfg Config) (jwtgo.SigningMethod, any, any, error) {
	switch cfg.Algorithm {
	case jwtAlgHS256:
		if strings.TrimSpace(cfg.Secret) == "" {
			return nil, nil, nil, ErrMissingSecret
		}
		key := []byte(cfg.Secret)
		return jwtgo.SigningMethodHS256, key, key, nil
	case jwtAlgEdDSA:
		privateKeyRaw, publicKeyRaw, err := resolveEdDSAKeyMaterial(cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		if strings.TrimSpace(privateKeyRaw) == "" {
			return nil, nil, nil, ErrMissingPrivateKey
		}
		if strings.TrimSpace(publicKeyRaw) == "" {
			return nil, nil, nil, ErrMissingPublicKey
		}
		privateKey, err := parseEd25519PrivateKey(privateKeyRaw)
		if err != nil {
			return nil, nil, nil, err
		}
		publicKey, err := parseEd25519PublicKey(publicKeyRaw)
		if err != nil {
			return nil, nil, nil, err
		}
		return jwtgo.SigningMethodEdDSA, privateKey, publicKey, nil
	default:
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrUnsupportedAlg, cfg.Algorithm)
	}
}

func resolveEdDSAKeyMaterial(cfg Config) (privateKeyRaw string, publicKeyRaw string, err error) {
	privateKeyRaw = cfg.PrivateKey
	publicKeyRaw = cfg.PublicKey

	privateKeyFile := strings.TrimSpace(cfg.PrivateKeyFile)
	if privateKeyFile != "" {
		privateKeyBytes, readErr := os.ReadFile(privateKeyFile)
		if readErr != nil {
			return "", "", fmt.Errorf("%w: %v", ErrReadPrivateKeyFile, readErr)
		}
		privateKeyRaw = string(privateKeyBytes)
	}

	publicKeyFile := strings.TrimSpace(cfg.PublicKeyFile)
	if publicKeyFile != "" {
		publicKeyBytes, readErr := os.ReadFile(publicKeyFile)
		if readErr != nil {
			return "", "", fmt.Errorf("%w: %v", ErrReadPublicKeyFile, readErr)
		}
		publicKeyRaw = string(publicKeyBytes)
	}

	return privateKeyRaw, publicKeyRaw, nil
}

func parseEd25519PrivateKey(raw string) (ed25519.PrivateKey, error) {
	if key, ok, err := parseEd25519PrivatePEM(raw); ok {
		if err != nil {
			return nil, err
		}
		return key, nil
	}

	keyBytes, err := decodeBase64(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}
	switch len(keyBytes) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(keyBytes), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(keyBytes), nil
	default:
		return nil, fmt.Errorf("%w: unexpected key size %d", ErrInvalidPrivateKey, len(keyBytes))
	}
}

func parseEd25519PublicKey(raw string) (ed25519.PublicKey, error) {
	if key, ok, err := parseEd25519PublicPEM(raw); ok {
		if err != nil {
			return nil, err
		}
		return key, nil
	}

	keyBytes, err := decodeBase64(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPublicKey, err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: unexpected key size %d", ErrInvalidPublicKey, len(keyBytes))
	}
	return ed25519.PublicKey(keyBytes), nil
}

func parseEd25519PrivatePEM(raw string) (ed25519.PrivateKey, bool, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil {
		return nil, false, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, true, fmt.Errorf("%w: parse pkcs8: %v", ErrInvalidPrivateKey, err)
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, true, fmt.Errorf("%w: expected ed25519 key", ErrInvalidPrivateKey)
	}
	return privateKey, true, nil
}

func parseEd25519PublicPEM(raw string) (ed25519.PublicKey, bool, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil {
		return nil, false, nil
	}

	if block.Type == "CERTIFICATE" {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, true, fmt.Errorf("%w: parse certificate: %v", ErrInvalidPublicKey, err)
		}
		publicKey, ok := cert.PublicKey.(ed25519.PublicKey)
		if !ok {
			return nil, true, fmt.Errorf("%w: certificate does not contain ed25519 key", ErrInvalidPublicKey)
		}
		return publicKey, true, nil
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, true, fmt.Errorf("%w: parse pkix: %v", ErrInvalidPublicKey, err)
	}
	publicKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, true, fmt.Errorf("%w: expected ed25519 key", ErrInvalidPublicKey)
	}
	return publicKey, true, nil
}

func decodeBase64(v string) ([]byte, error) {
	raw := strings.TrimSpace(v)
	encoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, enc := range encoders {
		out, err := enc.DecodeString(raw)
		if err == nil {
			return out, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
