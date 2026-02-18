package apikey

import (
	"context"
	"errors"
	"strings"
	"time"

	httpx "github.com/bronystylecrazy/ultrastructure/security/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

var (
	ErrLookupNotConfigured  = errors.New("apikey: key lookup is not configured")
	ErrRevokerNotConfigured = errors.New("apikey: revoker is not configured")
	ErrRotatorNotConfigured = errors.New("apikey: rotator is not configured")
	ErrInvalidAPIKey        = errors.New("apikey: invalid api key")
	ErrRevokedAPIKey        = errors.New("apikey: api key revoked")
	ErrExpiredAPIKey        = errors.New("apikey: api key expired")
)

type Service struct {
	config    Config
	generator Generator
	hasher    Hasher
	lookup    KeyLookup
	recorder  KeyUsageRecorder
	revoker   Revoker
	rotator   Rotator
	now       func() time.Time
}

var _ Manager = (*Service)(nil)

type NewServiceParams struct {
	Config    Config
	Generator Generator
	Hasher    Hasher
	Lookup    KeyLookup
	Recorder  KeyUsageRecorder
	Revoker   Revoker
	Rotator   Rotator
}

func NewService(p NewServiceParams) *Service {
	return &Service{
		config:    p.Config.withDefaults(),
		generator: p.Generator,
		hasher:    p.Hasher,
		lookup:    p.Lookup,
		recorder:  p.Recorder,
		revoker:   p.Revoker,
		rotator:   p.Rotator,
		now:       time.Now,
	}
}

func (s *Service) IssueKey(appID string, prefix string, scopes []string, metadata map[string]string, expiresAt *time.Time) (*IssuedKey, error) {
	raw, keyID, secret, err := s.generator.GenerateRawKey(prefix)
	if err != nil {
		return nil, err
	}
	hash, err := s.hasher.Hash(secret)
	if err != nil {
		return nil, err
	}
	return &IssuedKey{
		KeyID:      keyID,
		AppID:      appID,
		RawKey:     raw,
		Prefix:     s.resolvePrefix(prefix),
		SecretHash: hash,
		Scopes:     append([]string(nil), scopes...),
		Metadata:   cloneMap(metadata),
		ExpiresAt:  expiresAt,
	}, nil
}

func (s *Service) ValidateRawKey(ctx context.Context, rawKey string) (*Principal, error) {
	if s.lookup == nil {
		return nil, ErrLookupNotConfigured
	}
	keyID, secret, err := s.generator.ParseRawKey(rawKey)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	stored, err := s.lookup.FindByKeyID(ctx, keyID)
	if err != nil || stored == nil {
		return nil, ErrInvalidAPIKey
	}
	if stored.RevokedAt != nil {
		return nil, ErrRevokedAPIKey
	}
	now := s.now().UTC()
	if stored.ExpiresAt != nil && stored.ExpiresAt.Add(s.config.SkewAllowance).Before(now) {
		return nil, ErrExpiredAPIKey
	}
	ok, err := s.hasher.Verify(stored.SecretHash, secret)
	if err != nil || !ok {
		return nil, ErrInvalidAPIKey
	}

	if s.recorder != nil {
		_ = s.recorder.MarkUsed(ctx, stored.KeyID, now)
	}

	return &Principal{
		Type:     PrincipalTypeAPIKey,
		AppID:    stored.AppID,
		KeyID:    stored.KeyID,
		Scopes:   append([]string(nil), stored.Scopes...),
		Metadata: cloneMap(stored.Metadata),
	}, nil
}

func (s *Service) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := s.extractRawKey(c)
		if raw == "" {
			return s.unauthorized(c, ErrInvalidAPIKey)
		}
		principal, err := s.ValidateRawKey(c.Context(), raw)
		if err != nil {
			return s.unauthorized(c, err)
		}
		if s.config.SetPrincipalBody {
			SetPrincipalLocals(c, principal)
		}
		if s.config.SetPrincipalCtx {
			c.SetContext(WithPrincipal(c.Context(), principal))
		}
		return c.Next()
	}
}

func (s *Service) unauthorized(c fiber.Ctx, err error) error {
	msg := ErrInvalidAPIKey.Error()
	if s.config.DetailedErrors {
		msg = err.Error()
	}
	return httpx.Unauthorized(c, msg)
}

func (s *Service) RevokeKey(ctx context.Context, keyID string, reason string) error {
	if s.revoker == nil {
		return ErrRevokerNotConfigured
	}
	return s.revoker.RevokeKey(ctx, keyID, reason)
}

func (s *Service) RotateKey(ctx context.Context, keyID string, prefix string) (*IssuedKey, error) {
	if s.rotator == nil {
		return nil, ErrRotatorNotConfigured
	}
	return s.rotator.RotateKey(ctx, keyID, prefix)
}

func (s *Service) extractRawKey(c fiber.Ctx) string {
	header := strings.TrimSpace(c.Get(s.config.HeaderName))
	if strings.EqualFold(s.config.HeaderName, "Authorization") {
		if header == "" {
			return strings.TrimSpace(c.Get("X-API-Key"))
		}
		if s.config.HeaderScheme == "" {
			return header
		}
		scheme := s.config.HeaderScheme + " "
		if strings.HasPrefix(strings.ToLower(header), strings.ToLower(scheme)) {
			return strings.TrimSpace(header[len(scheme):])
		}
		return ""
	}
	return header
}

func (s *Service) resolvePrefix(prefix string) string {
	p := strings.TrimSpace(prefix)
	if p != "" {
		return p
	}
	return s.config.KeyPrefix
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
