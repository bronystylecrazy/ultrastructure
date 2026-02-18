package token

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bronystylecrazy/ultrastructure/caching/rd"
	"github.com/gofiber/fiber/v3"
	redis "github.com/redis/go-redis/v9"
)

const defaultRevocationKeyPrefix = "token:revoked:"
const defaultRevocationNamespace = "default"

var (
	ErrRevocationStoreNotConfigured = fmt.Errorf("token: revocation store not configured")
	ErrMissingTokenJTI              = fmt.Errorf("token: missing jti in token")
	ErrMissingTokenExp              = fmt.Errorf("token: missing exp in token")
	ErrTokenRevoked                 = fmt.Errorf("token: token revoked")
)

type RevocationStore interface {
	Revoke(ctx context.Context, jti string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

type RedisRevocationStore struct {
	client    rd.StringManager
	keyPrefix string
	namespace string
	now       func() time.Time
}

func NewRedisRevocationStore(client rd.StringManager, keyPrefix string) *RedisRevocationStore {
	return NewRedisRevocationStoreWithNamespace(client, keyPrefix, "")
}

func NewRedisRevocationStoreWithNamespace(client rd.StringManager, keyPrefix string, namespace string) *RedisRevocationStore {
	if keyPrefix == "" {
		keyPrefix = defaultRevocationKeyPrefix
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = defaultRevocationNamespace
	}
	return &RedisRevocationStore{
		client:    client,
		keyPrefix: keyPrefix,
		namespace: ns,
		now:       time.Now,
	}
}

func (r *RedisRevocationStore) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return r.client.Set(ctx, r.key(jti), "1", ttl).Err()
}

func (r *RedisRevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	_, err := r.client.Get(ctx, r.key(jti)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *RedisRevocationStore) key(jti string) string {
	if r.namespace != "" {
		return r.keyPrefix + r.namespace + ":" + jti
	}
	return r.keyPrefix + jti
}

func (s *Service) SetRevocationStore(store RevocationStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revocationStore = store
}

func (s *Service) revocation() RevocationStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revocationStore
}

func (s *Service) RevokeToken(ctx context.Context, tokenValue string) error {
	claims, err := s.ValidateToken(tokenValue, "")
	if err != nil {
		return err
	}
	return s.RevokeClaims(ctx, claims)
}

func (s *Service) RevokeClaims(ctx context.Context, claims Claims) error {
	store := s.revocation()
	if store == nil {
		return ErrRevocationStoreNotConfigured
	}

	if claims.JTI == "" {
		return ErrMissingTokenJTI
	}
	if claims.ExpiresAt.IsZero() {
		return ErrMissingTokenExp
	}
	return store.Revoke(ctx, claims.JTI, claims.ExpiresAt)
}

func (s *Service) RevokeFromContext(c fiber.Ctx) error {
	claims, err := ClaimsFromContext(c)
	if err != nil {
		return err
	}
	return s.RevokeClaims(c.Context(), claims)
}

func (s *Service) ensureNotRevoked(ctx context.Context, claims Claims) error {
	store := s.revocation()
	if store == nil {
		return nil
	}
	if claims.JTI == "" {
		return ErrMissingTokenJTI
	}
	revoked, err := store.IsRevoked(ctx, claims.JTI)
	if err != nil {
		return err
	}
	if revoked {
		return ErrTokenRevoked
	}
	return nil
}
