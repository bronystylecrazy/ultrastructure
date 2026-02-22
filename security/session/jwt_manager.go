package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bronystylecrazy/ultrastructure/security/jws"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const claimsContextKey = "us.session.claims"

const defaultAccessTokenTTL = 15 * time.Minute
const defaultRefreshTokenTTL = 720 * time.Hour

type JWTManager struct {
	config jws.Config
	signer jws.SignerVerifier
	now    func() time.Time

	mu                      sync.RWMutex
	defaultAccessExtractor  Extractor
	defaultRefreshExtractor Extractor
	revocationStore         RevocationStore
}

var _ Manager = (*JWTManager)(nil)

func NewJWTManager(config jws.Config, signer jws.SignerVerifier) (*JWTManager, error) {
	if signer == nil {
		return nil, ErrSignerNotConfigured
	}
	if config.AccessTokenTTL <= 0 {
		config.AccessTokenTTL = defaultAccessTokenTTL
	}
	if config.RefreshTokenTTL <= 0 {
		config.RefreshTokenTTL = defaultRefreshTokenTTL
	}
	return &JWTManager{
		config:                  config,
		signer:                  signer,
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

func (s *JWTManager) SetDefaultAccessExtractors(exs ...Extractor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(exs) == 0 {
		s.defaultAccessExtractor = defaultAccessExtractor()
		return
	}
	s.defaultAccessExtractor = Chain(exs...)
}

func (s *JWTManager) SetDefaultRefreshExtractors(exs ...Extractor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(exs) == 0 {
		s.defaultRefreshExtractor = defaultRefreshExtractor()
		return
	}
	s.defaultRefreshExtractor = Chain(exs...)
}

func (s *JWTManager) defaultAccess() Extractor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultAccessExtractor
}

func (s *JWTManager) defaultRefresh() Extractor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultRefreshExtractor
}

func (s *JWTManager) Generate(subject string, opts ...GenerateOption) (*TokenPair, error) {
	cfg := resolveGenerateConfig(opts...)

	now := s.now().UTC()
	accessExp := now.Add(s.config.AccessTokenTTL)
	refreshExp := now.Add(s.config.RefreshTokenTTL)

	accessToken, err := s.signToken(subject, TokenTypeAccess, accessExp, cfg.AccessClaims)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.signToken(subject, TokenTypeRefresh, refreshExp, cfg.RefreshClaims)
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

func (s *JWTManager) RotateRefresh(refreshToken string, opts ...GenerateOption) (*TokenPair, error) {
	claims, err := s.Validate(refreshToken, TokenTypeRefresh)
	if err != nil {
		return nil, err
	}
	if claims.Subject == "" {
		return nil, ErrMissingTokenSub
	}
	if err := s.RevokeClaims(context.Background(), claims); err != nil {
		return nil, err
	}
	return s.Generate(claims.Subject, opts...)
}

func (s *JWTManager) RotateAccess(accessToken string, opts ...GenerateOption) (string, time.Time, error) {
	cfg := resolveGenerateConfig(opts...)

	claims, err := s.Validate(accessToken, TokenTypeAccess)
	if err != nil {
		return "", time.Time{}, err
	}
	if claims.Subject == "" {
		return "", time.Time{}, ErrMissingTokenSub
	}
	if err := s.RevokeClaims(context.Background(), claims); err != nil {
		return "", time.Time{}, err
	}

	expiresAt := s.now().UTC().Add(s.config.AccessTokenTTL)
	token, err := s.signToken(claims.Subject, TokenTypeAccess, expiresAt, cfg.AccessClaims)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *JWTManager) Validate(tokenValue string, expectedType string) (Claims, error) {
	out, err := s.signer.Verify(tokenValue)
	if err != nil {
		return Claims{}, err
	}
	claims := fromJWSClaims(out)
	if claims.TokenType == "" {
		return Claims{}, ErrInvalidTokenType
	}
	if expectedType != "" && claims.TokenType != expectedType {
		return Claims{}, fmt.Errorf("%w: got=%s want=%s", ErrInvalidTokenType, claims.TokenType, expectedType)
	}

	return claims, nil
}

func (s *JWTManager) AccessMiddleware(exs ...Extractor) fiber.Handler {
	if len(exs) == 0 {
		return s.tokenMiddleware(TokenTypeAccess, s.defaultAccess())
	}
	return s.tokenMiddleware(TokenTypeAccess, chainOrDefault(exs...))
}

func (s *JWTManager) RefreshMiddleware(exs ...Extractor) fiber.Handler {
	if len(exs) == 0 {
		return s.tokenMiddleware(TokenTypeRefresh, s.defaultRefresh())
	}
	return s.tokenMiddleware(TokenTypeRefresh, chainOrDefault(exs...))
}

func (s *JWTManager) tokenMiddleware(expectedType string, extractor Extractor) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenValue, err := extractor.Extract(c)
		if err != nil {
			return writeUnauthorized(c, err)
		}
		if tokenValue == "" {
			return writeUnauthorized(c, ErrTokenMissingInContext)
		}

		claims, err := s.Validate(tokenValue, expectedType)
		if err != nil {
			return writeUnauthorized(c, err)
		}
		if err := s.ensureNotRevoked(c.Context(), claims); err != nil {
			return writeUnauthorized(c, err)
		}

		c.Locals(claimsContextKey, claims)
		return c.Next()
	}
}

func ClaimsFromContext(c fiber.Ctx) (Claims, error) {
	raw := c.Locals(claimsContextKey)
	if raw == nil {
		return Claims{}, ErrTokenMissingInContext
	}
	claims, ok := raw.(Claims)
	if !ok {
		return Claims{}, ErrInvalidClaims
	}
	return claims, nil
}

func SubjectFromContext(c fiber.Ctx) (string, error) {
	claims, err := ClaimsFromContext(c)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", ErrMissingTokenSub
	}
	return claims.Subject, nil
}

func chainOrDefault(exs ...Extractor) Extractor {
	if len(exs) == 0 {
		return FromAuthHeader("Bearer")
	}
	return Chain(exs...)
}

func defaultAccessExtractor() Extractor {
	return Chain(
		FromAuthHeader("Bearer"),
		FromHeader("X-Access-Token"),
		FromCookie("access_token"),
		FromCookie("token"),
		FromQuery("access_token"),
		FromQuery("token"),
		FromForm("access_token"),
		FromForm("token"),
		FromParam("access_token"),
		FromParam("token"),
	)
}

func defaultRefreshExtractor() Extractor {
	return Chain(
		FromAuthHeader("Bearer"),
		FromHeader("X-Refresh-Token"),
		FromCookie("refresh_token"),
		FromCookie("token"),
		FromQuery("refresh_token"),
		FromQuery("token"),
		FromForm("refresh_token"),
		FromForm("token"),
		FromParam("refresh_token"),
		FromParam("token"),
	)
}

func (s *JWTManager) signToken(subject, tokenType string, expiresAt time.Time, customClaims map[string]any) (string, error) {
	now := s.now().UTC().Unix()
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
	return s.signer.Sign(claims)
}

func fromJWSClaims(in jws.Claims) Claims {
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
