package paseto

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/o1egl/paseto"
	"golang.org/x/crypto/chacha20poly1305"
)

// Paseto implements SignerVerifier using the o1egl/paseto library.
type Paseto struct {
	config    Config
	v2        *paseto.V2
	symmetric []byte
	now       func() time.Time
}

var _ SignerVerifier = (*Paseto)(nil)

// New creates a new PASETO SignerVerifier.
func New(config Config) (*Paseto, error) {
	cfg := config.withDefaults()

	// Validate and setup the symmetric key
	var symmetricKey []byte
	if cfg.Version == "v2" {
		// For V2, we need a 32-byte key for ChaCha20-Poly1305
		secretBytes := []byte(cfg.Secret)
		if len(secretBytes) < chacha20poly1305.KeySize {
			// Derive a proper key from the secret using a simple hash approach
			symmetricKey = deriveKey(cfg.Secret)
		} else {
			symmetricKey = secretBytes[:chacha20poly1305.KeySize]
		}
	} else {
		// V1 uses HMAC-SHA384, key can be any length
		symmetricKey = []byte(cfg.Secret)
	}

	return &Paseto{
		config:    cfg,
		v2:        paseto.NewV2(),
		symmetric: symmetricKey,
		now:       time.Now,
	}, nil
}

// deriveKey derives a 32-byte key from a secret string.
func deriveKey(secret string) []byte {
	// Use a simple approach: if the secret is too short, expand it
	key := make([]byte, chacha20poly1305.KeySize)
	secretBytes := []byte(secret)

	if len(secretBytes) >= chacha20poly1305.KeySize {
		copy(key, secretBytes[:chacha20poly1305.KeySize])
		return key
	}

	// Expand the key by repeating and adding a salt
	for i := 0; i < chacha20poly1305.KeySize; i++ {
		key[i] = secretBytes[i%len(secretBytes)] ^ byte(i)
	}

	return key
}

// Sign signs a PASETO token with the given claims.
func (p *Paseto) Sign(claims map[string]any) (string, error) {
	now := p.now().UTC()

	// Ensure standard claims are set
	if _, ok := claims["jti"]; !ok {
		claims["jti"] = uuid.NewString()
	}
	if _, ok := claims["iat"]; !ok {
		claims["iat"] = now.Unix()
	}
	if _, ok := claims["nbf"]; !ok {
		claims["nbf"] = now.Unix()
	}
	if p.config.Issuer != "" {
		if _, ok := claims["iss"]; !ok {
			claims["iss"] = p.config.Issuer
		}
	}

	// PASETO V2 Encrypt expects a struct that can be JSON-encoded
	token, err := p.v2.Encrypt(p.symmetric, claims, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	return token, nil
}

// Verify verifies a PASETO token and returns the claims.
func (p *Paseto) Verify(tokenValue string) (Claims, error) {
	// Check token version prefix
	if len(tokenValue) < 4 {
		return Claims{}, fmt.Errorf("%w: token too short", ErrInvalidToken)
	}

	version := tokenValue[:2]
	expectedVersion := "v2"
	if p.config.Version != "" {
		expectedVersion = p.config.Version
	}

	if version != expectedVersion {
		return Claims{}, fmt.Errorf("%w: got=%s want=%s", ErrUnexpectedTokenVersion, version, expectedVersion)
	}

	// Decrypt and verify the token
	// The o1egl/paseto library expects a pointer to a struct/map for the output
	var decrypted map[string]interface{}
	err := p.v2.Decrypt(tokenValue, p.symmetric, &decrypted, nil)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Check expiration
	now := p.now().UTC().Unix()
	if exp, ok := decrypted["exp"].(float64); ok {
		if now > int64(exp) {
			return Claims{}, ErrExpiredToken
		}
	}

	// Convert map[string]interface{} to map[string]any
	jsonClaims := make(map[string]any, len(decrypted))
	for k, v := range decrypted {
		jsonClaims[k] = v
	}

	return fromMapClaims(jsonClaims), nil
}

// GenerateKey generates a new random symmetric key suitable for PASETO V2.
func GenerateKey() (string, error) {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// ClaimsFromContext extracts PASETO claims from a Fiber context.
func ClaimsFromContext(c fiber.Ctx) (Claims, error) {
	raw := c.Locals("paseto_claims")
	if raw == nil {
		return Claims{}, ErrInvalidClaims
	}
	claims, ok := raw.(Claims)
	if !ok {
		return Claims{}, ErrInvalidClaims
	}
	return claims, nil
}

// SubjectFromContext extracts the subject from PASETO claims in the context.
func SubjectFromContext(c fiber.Ctx) (string, error) {
	claims, err := ClaimsFromContext(c)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", ErrInvalidClaims
	}
	return claims.Subject, nil
}

// MarshalJSON implements json.Marshaler for claims conversion.
func marshalClaims(claims map[string]any) ([]byte, error) {
	return json.Marshal(claims)
}
