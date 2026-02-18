package token

import (
	"errors"
	"fmt"
	"sync"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrMissingSecret      = errors.New("token: missing secret")
	ErrInvalidClaims      = errors.New("token: invalid claims")
	ErrInvalidTokenType   = errors.New("token: invalid token type")
	ErrMissingTokenSub    = errors.New("token: missing subject in token")
	ErrUnexpectedTokenAlg = errors.New("token: unexpected signing method")
)

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type Service struct {
	config     Config
	signingKey []byte
	now        func() time.Time

	mu                      sync.RWMutex
	defaultAccessExtractor  Extractor
	defaultRefreshExtractor Extractor
	revocationStore         RevocationStore
}

var _ Manager = (*Service)(nil)

func NewService(config Config) (*Service, error) {
	cfg := config.withDefaults()
	if cfg.Secret == "" {
		return nil, ErrMissingSecret
	}
	return &Service{
		config:                  cfg,
		signingKey:              []byte(cfg.Secret),
		now:                     time.Now,
		defaultAccessExtractor:  FromAuthHeader("Bearer"),
		defaultRefreshExtractor: FromAuthHeader("Bearer"),
	}, nil
}

func (s *Service) SetDefaultAccessExtractors(exs ...Extractor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultAccessExtractor = chainOrDefault(exs...)
}

func (s *Service) SetDefaultRefreshExtractors(exs ...Extractor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultRefreshExtractor = chainOrDefault(exs...)
}

func (s *Service) defaultAccess() Extractor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultAccessExtractor
}

func (s *Service) defaultRefresh() Extractor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultRefreshExtractor
}

func (s *Service) GenerateTokenPair(subject string, additionalAccessClaims map[string]any) (*TokenPair, error) {
	now := s.now().UTC()
	accessExp := now.Add(s.config.AccessTokenTTL)
	refreshExp := now.Add(s.config.RefreshTokenTTL)

	accessToken, err := s.signToken(subject, TokenTypeAccess, accessExp, additionalAccessClaims)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.signToken(subject, TokenTypeRefresh, refreshExp, nil)
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

func (s *Service) RefreshTokenPair(refreshToken string, additionalAccessClaims map[string]any) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken, TokenTypeRefresh)
	if err != nil {
		return nil, err
	}

	if claims.Subject == "" {
		return nil, ErrMissingTokenSub
	}

	return s.GenerateTokenPair(claims.Subject, additionalAccessClaims)
}

func (s *Service) ValidateToken(tokenValue string, expectedType string) (Claims, error) {
	token, err := jwtgo.Parse(tokenValue, func(token *jwtgo.Token) (any, error) {
		if _, ok := token.Method.(*jwtgo.SigningMethodHMAC); !ok {
			return nil, ErrUnexpectedTokenAlg
		}
		return s.signingKey, nil
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

	out := claimsFromJWT(claims)
	if out.TokenType == "" {
		return Claims{}, ErrInvalidTokenType
	}
	if expectedType != "" && out.TokenType != expectedType {
		return Claims{}, fmt.Errorf("%w: got=%s want=%s", ErrInvalidTokenType, out.TokenType, expectedType)
	}

	return out, nil
}

func (s *Service) signToken(subject, tokenType string, expiresAt time.Time, customClaims map[string]any) (string, error) {
	now := s.now().UTC()
	claims := jwtgo.MapClaims{
		"sub":        subject,
		"jti":        uuid.NewString(),
		"iat":        now.Unix(),
		"nbf":        now.Unix(),
		"exp":        expiresAt.Unix(),
		"token_type": tokenType,
	}
	if s.config.Issuer != "" {
		claims["iss"] = s.config.Issuer
	}
	for k, v := range customClaims {
		claims[k] = v
	}

	t := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, claims)
	return t.SignedString(s.signingKey)
}
