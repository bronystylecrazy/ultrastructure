package session

import (
	"context"
	"sync"
	"time"

	"github.com/bronystylecrazy/ultrastructure/x/paseto"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// PasetoManager implements session.Manager using PASETO tokens.
type PasetoManager struct {
	config                  paseto.Config
	paseto                  paseto.SignerVerifier
	now                     func() time.Time
	mu                      sync.RWMutex
	defaultAccessExtractor  Extractor
	defaultRefreshExtractor Extractor
	revocationStore         RevocationStore
}

var _ Manager = (*PasetoManager)(nil)

// NewPasetoManager creates a new PASETO-based session manager.
func NewPasetoManager(config paseto.Config, pv paseto.SignerVerifier) (*PasetoManager, error) {
	if pv == nil {
		return nil, ErrSignerNotConfigured
	}
	if config.AccessTokenTTL <= 0 {
		config.AccessTokenTTL = defaultAccessTokenTTL
	}
	if config.RefreshTokenTTL <= 0 {
		config.RefreshTokenTTL = defaultRefreshTokenTTL
	}
	return &PasetoManager{
		config:                  config,
		paseto:                  pv,
		now:                     time.Now,
		defaultAccessExtractor:  defaultAccessExtractor(),
		defaultRefreshExtractor: defaultRefreshExtractor(),
		revocationStore: NewRevocationStoreWithNamespace(
			NewInMemoryRevocationCache(),
			"",
			config.Issuer,
		),
	}, nil
}

// SetDefaultAccessExtractors sets the default extractors for access tokens.
func (m *PasetoManager) SetDefaultAccessExtractors(exs ...Extractor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(exs) == 0 {
		m.defaultAccessExtractor = defaultAccessExtractor()
		return
	}
	m.defaultAccessExtractor = Chain(exs...)
}

// SetDefaultRefreshExtractors sets the default extractors for refresh tokens.
func (m *PasetoManager) SetDefaultRefreshExtractors(exs ...Extractor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(exs) == 0 {
		m.defaultRefreshExtractor = defaultRefreshExtractor()
		return
	}
	m.defaultRefreshExtractor = Chain(exs...)
}

func (m *PasetoManager) defaultAccess() Extractor {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultAccessExtractor
}

func (m *PasetoManager) defaultRefresh() Extractor {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultRefreshExtractor
}

// Generate creates a new token pair.
func (m *PasetoManager) Generate(subject string, opts ...GenerateOption) (*TokenPair, error) {
	cfg := resolveGenerateConfig(opts...)

	now := m.now().UTC()
	accessExp := now.Add(m.config.AccessTokenTTL)
	refreshExp := now.Add(m.config.RefreshTokenTTL)

	accessToken, err := m.signToken(subject, TokenTypeAccess, accessExp, cfg.AccessClaims)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.signToken(subject, TokenTypeRefresh, refreshExp, cfg.RefreshClaims)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}

// RotateRefresh invalidates the old refresh token and returns a new token pair.
func (m *PasetoManager) RotateRefresh(refreshToken string, opts ...GenerateOption) (*TokenPair, error) {
	claims, err := m.Validate(refreshToken, TokenTypeRefresh)
	if err != nil {
		return nil, err
	}
	if claims.Subject == "" {
		return nil, ErrMissingTokenSub
	}
	if err := m.RevokeClaims(context.Background(), claims); err != nil {
		return nil, err
	}
	return m.Generate(claims.Subject, opts...)
}

// RotateAccess invalidates the old access token and returns a new one.
func (m *PasetoManager) RotateAccess(accessToken string, opts ...GenerateOption) (string, time.Time, error) {
	cfg := resolveGenerateConfig(opts...)

	claims, err := m.Validate(accessToken, TokenTypeAccess)
	if err != nil {
		return "", time.Time{}, err
	}
	if claims.Subject == "" {
		return "", time.Time{}, ErrMissingTokenSub
	}
	if err := m.RevokeClaims(context.Background(), claims); err != nil {
		return "", time.Time{}, err
	}

	expiresAt := m.now().UTC().Add(m.config.AccessTokenTTL)
	token, err := m.signToken(claims.Subject, TokenTypeAccess, expiresAt, cfg.AccessClaims)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// Validate verifies a token and returns its claims.
func (m *PasetoManager) Validate(tokenValue string, expectedType string) (Claims, error) {
	out, err := m.paseto.Verify(tokenValue)
	if err != nil {
		return Claims{}, err
	}
	claims := fromPasetoClaims(out)
	if claims.TokenType == "" {
		return Claims{}, ErrInvalidTokenType
	}
	if expectedType != "" && claims.TokenType != expectedType {
		return Claims{}, ErrInvalidTokenType
	}

	return claims, nil
}

// AccessMiddleware creates middleware for access token authentication.
func (m *PasetoManager) AccessMiddleware(exs ...Extractor) fiber.Handler {
	if len(exs) == 0 {
		return m.tokenMiddleware(TokenTypeAccess, m.defaultAccess())
	}
	return m.tokenMiddleware(TokenTypeAccess, chainOrDefault(exs...))
}

// RefreshMiddleware creates middleware for refresh token authentication.
func (m *PasetoManager) RefreshMiddleware(exs ...Extractor) fiber.Handler {
	if len(exs) == 0 {
		return m.tokenMiddleware(TokenTypeRefresh, m.defaultRefresh())
	}
	return m.tokenMiddleware(TokenTypeRefresh, chainOrDefault(exs...))
}

func (m *PasetoManager) tokenMiddleware(expectedType string, extractor Extractor) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenValue, err := extractor.Extract(c)
		if err != nil {
			return writeUnauthorized(c, err)
		}
		if tokenValue == "" {
			return writeUnauthorized(c, ErrTokenMissingInContext)
		}

		claims, err := m.Validate(tokenValue, expectedType)
		if err != nil {
			return writeUnauthorized(c, err)
		}
		if err := m.ensureNotRevoked(c.Context(), claims); err != nil {
			return writeUnauthorized(c, err)
		}

		c.Locals(claimsContextKey, claims)
		return c.Next()
	}
}

// Revoke revokes a token by its value.
func (m *PasetoManager) Revoke(ctx context.Context, tokenValue string) error {
	claims, err := m.Validate(tokenValue, "")
	if err != nil {
		return err
	}
	return m.RevokeClaims(ctx, claims)
}

// RevokeFromContext revokes the token from the current context.
func (m *PasetoManager) RevokeFromContext(c fiber.Ctx) error {
	claims, err := ClaimsFromContext(c)
	if err != nil {
		return err
	}
	return m.RevokeClaims(c.Context(), claims)
}

// RevokeClaims revokes token claims.
func (m *PasetoManager) RevokeClaims(ctx context.Context, claims Claims) error {
	if m.revocationStore == nil {
		return nil
	}
	return m.revocationStore.Revoke(ctx, claims.JTI, claims.ExpiresAt)
}

func (m *PasetoManager) ensureNotRevoked(ctx context.Context, claims Claims) error {
	if m.revocationStore == nil {
		return nil
	}
	revoked, err := m.revocationStore.IsRevoked(ctx, claims.JTI)
	if err != nil {
		return err
	}
	if revoked {
		return ErrTokenRevoked
	}
	return nil
}

// SetRevocationStore sets the revocation store.
func (m *PasetoManager) SetRevocationStore(store RevocationStore) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revocationStore = store
}

func (m *PasetoManager) signToken(subject, tokenType string, expiresAt time.Time, customClaims map[string]any) (string, error) {
	now := m.now().UTC().Unix()
	claims := map[string]any{
		"sub": subject,
		"jti": uuid.NewString(),
		"iat": now,
		"nbf": now,
		"exp": expiresAt.Unix(),
		"typ": tokenType,
	}
	for k, v := range customClaims {
		claims[k] = v
	}
	return m.paseto.Sign(claims)
}

func fromPasetoClaims(in paseto.Claims) Claims {
	values := make(map[string]any, len(in.Values))
	for k, v := range in.Values {
		values[k] = v
	}
	return Claims{
		Subject:   in.Subject,
		TokenType: in.TokenType,
		JTI:       in.JTI,
		ExpiresAt: in.ExpiresAt,
		Values:    values,
	}
}
