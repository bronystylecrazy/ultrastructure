package session

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type inMemoryRevocationEntry struct {
	value     string
	expiresAt time.Time
}

type inMemoryRevocationCache struct {
	mu    sync.RWMutex
	items map[string]inMemoryRevocationEntry
}

func NewInMemoryRevocationCache() RevocationCache {
	return &inMemoryRevocationCache{
		items: make(map[string]inMemoryRevocationEntry),
	}
}

func (c *inMemoryRevocationCache) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	c.mu.Lock()
	c.items[key] = inMemoryRevocationEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
	return nil
}

func (c *inMemoryRevocationCache) Get(_ context.Context, key string) (string, error) {
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return "", ErrRevocationCacheMiss
	}
	if now.After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return "", ErrRevocationCacheMiss
	}
	return entry.value, nil
}

type RevocationStoreImpl struct {
	client    RevocationCache
	keyPrefix string
	namespace string
}

func NewRevocationStore(client RevocationCache, keyPrefix string) *RevocationStoreImpl {
	return NewRevocationStoreWithNamespace(client, keyPrefix, "")
}

func NewRevocationStoreWithNamespace(client RevocationCache, keyPrefix string, namespace string) *RevocationStoreImpl {
	if keyPrefix == "" {
		keyPrefix = DefaultRevocationKeyPrefix
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = DefaultRevocationNamespace
	}
	return &RevocationStoreImpl{
		client:    client,
		keyPrefix: keyPrefix,
		namespace: ns,
	}
}

func (r *RevocationStoreImpl) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return r.client.Set(ctx, r.key(jti), "1", ttl)
}

func (r *RevocationStoreImpl) IsRevoked(ctx context.Context, jti string) (bool, error) {
	_, err := r.client.Get(ctx, r.key(jti))
	if err != nil {
		if errors.Is(err, ErrRevocationCacheMiss) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *RevocationStoreImpl) key(jti string) string {
	if r.namespace != "" {
		return r.keyPrefix + r.namespace + ":" + jti
	}
	return r.keyPrefix + jti
}

func (s *JWTManager) SetRevocationStore(store RevocationStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revocationStore = store
}

func (s *JWTManager) revocation() RevocationStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revocationStore
}

func (s *JWTManager) Revoke(ctx context.Context, tokenValue string) error {
	claims, err := s.Validate(tokenValue, "")
	if err != nil {
		return err
	}
	return s.RevokeClaims(ctx, claims)
}

func (s *JWTManager) RevokeClaims(ctx context.Context, claims Claims) error {
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

func (s *JWTManager) RevokeFromContext(c fiber.Ctx) error {
	claims, err := ClaimsFromContext(c)
	if err != nil {
		return err
	}
	return s.RevokeClaims(c.Context(), claims)
}

func (s *JWTManager) ensureNotRevoked(ctx context.Context, claims Claims) error {
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
