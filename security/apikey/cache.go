package apikey

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultL1TTL          = 60 * time.Second
	defaultL2TTL          = 10 * time.Minute
	defaultNegativeTTL    = 30 * time.Second
	defaultL1MaxEntries   = 4096
	defaultRedisKeyPrefix = "apikey:lookup:"
	negativeMarker        = "__nil__"
)

var ErrCacheMiss = errors.New("apikey cache miss")

type CachedLookupConfig struct {
	L1TTL          time.Duration
	L2TTL          time.Duration
	NegativeTTL    time.Duration
	L1MaxEntries   int
	RedisKeyPrefix string
	Namespace      string
}

func (c CachedLookupConfig) withDefaults() CachedLookupConfig {
	if c.L1TTL <= 0 {
		c.L1TTL = defaultL1TTL
	}
	if c.L2TTL <= 0 {
		c.L2TTL = defaultL2TTL
	}
	if c.NegativeTTL <= 0 {
		c.NegativeTTL = defaultNegativeTTL
	}
	if c.L1MaxEntries <= 0 {
		c.L1MaxEntries = defaultL1MaxEntries
	}
	if c.RedisKeyPrefix == "" {
		c.RedisKeyPrefix = defaultRedisKeyPrefix
	}
	return c
}

type cacheEntry struct {
	key    *StoredKey
	expire time.Time
}

type LookupInvalidator interface {
	InvalidateKey(ctx context.Context, keyID string) error
}

type CacheStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type CachedLookup struct {
	base  KeyLookup
	redis CacheStore
	cfg   CachedLookupConfig
	now   func() time.Time

	mu sync.RWMutex
	l1 map[string]cacheEntry

	sf singleflight.Group
}

var _ LookupInvalidator = (*CachedLookup)(nil)

func NewCachedLookup(base KeyLookup, cacheStore CacheStore, cfg CachedLookupConfig) *CachedLookup {
	cfg = cfg.withDefaults()
	return &CachedLookup{
		base:  base,
		redis: cacheStore,
		cfg:   cfg,
		now:   time.Now,
		l1:    make(map[string]cacheEntry, cfg.L1MaxEntries),
	}
}

func (c *CachedLookup) FindByKeyID(ctx context.Context, keyID string) (*StoredKey, error) {
	if keyID == "" {
		return nil, nil
	}
	if v, ok := c.getL1(keyID); ok {
		return cloneStoredKey(v), nil
	}

	v, err, _ := c.sf.Do(keyID, func() (any, error) {
		if v, ok := c.getL1(keyID); ok {
			return cloneStoredKey(v), nil
		}

		if c.redis != nil {
			if v, found, err := c.getL2(ctx, keyID); err != nil {
				return nil, err
			} else if found {
				c.setL1(keyID, v, c.ttlFor(v))
				return cloneStoredKey(v), nil
			}
		}

		v, err := c.base.FindByKeyID(ctx, keyID)
		if err != nil {
			return nil, err
		}
		c.setL1(keyID, v, c.ttlFor(v))
		if c.redis != nil {
			if err := c.setL2(ctx, keyID, v, c.ttlFor(v)); err != nil {
				return nil, err
			}
		}
		return cloneStoredKey(v), nil
	})
	if err != nil {
		return nil, err
	}
	out, _ := v.(*StoredKey)
	return out, nil
}

func (c *CachedLookup) InvalidateKey(ctx context.Context, keyID string) error {
	c.mu.Lock()
	delete(c.l1, keyID)
	c.mu.Unlock()
	c.sf.Forget(keyID)
	if c.redis == nil {
		return nil
	}
	if err := c.redis.Del(ctx, c.redisKey(keyID)); err != nil && !errors.Is(err, ErrCacheMiss) {
		return err
	}
	return nil
}

func (c *CachedLookup) getL1(keyID string) (*StoredKey, bool) {
	now := c.now().UTC()
	c.mu.RLock()
	entry, ok := c.l1[keyID]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if now.After(entry.expire) {
		c.mu.Lock()
		delete(c.l1, keyID)
		c.mu.Unlock()
		return nil, false
	}
	return cloneStoredKey(entry.key), true
}

func (c *CachedLookup) setL1(keyID string, key *StoredKey, ttl time.Duration) {
	now := c.now().UTC()
	c.mu.Lock()
	if len(c.l1) >= c.cfg.L1MaxEntries {
		// very lightweight bounded map behavior: evict one arbitrary item.
		for k := range c.l1 {
			delete(c.l1, k)
			break
		}
	}
	c.l1[keyID] = cacheEntry{
		key:    cloneStoredKey(key),
		expire: now.Add(ttl),
	}
	c.mu.Unlock()
}

func (c *CachedLookup) getL2(ctx context.Context, keyID string) (*StoredKey, bool, error) {
	raw, err := c.redis.Get(ctx, c.redisKey(keyID))
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if raw == negativeMarker {
		return nil, true, nil
	}
	var out StoredKey
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		_ = c.redis.Del(ctx, c.redisKey(keyID))
		return nil, false, nil
	}
	return &out, true, nil
}

func (c *CachedLookup) setL2(ctx context.Context, keyID string, key *StoredKey, ttl time.Duration) error {
	var payload string
	if key == nil {
		payload = negativeMarker
	} else {
		b, err := json.Marshal(key)
		if err != nil {
			return err
		}
		payload = string(b)
	}
	return c.redis.Set(ctx, c.redisKey(keyID), payload, ttl)
}

func (c *CachedLookup) ttlFor(key *StoredKey) time.Duration {
	if key == nil {
		return c.cfg.NegativeTTL
	}
	now := c.now().UTC()
	ttl := c.cfg.L1TTL
	if key.ExpiresAt != nil {
		untilExp := key.ExpiresAt.Sub(now)
		if untilExp <= 0 {
			return c.cfg.NegativeTTL
		}
		if untilExp < ttl {
			ttl = untilExp
		}
	}
	if ttl <= 0 {
		ttl = c.cfg.NegativeTTL
	}
	return ttl
}

func (c *CachedLookup) redisKey(keyID string) string {
	if ns := strings.TrimSpace(c.cfg.Namespace); ns != "" {
		return c.cfg.RedisKeyPrefix + ns + ":" + keyID
	}
	return c.cfg.RedisKeyPrefix + keyID
}

func cloneStoredKey(in *StoredKey) *StoredKey {
	if in == nil {
		return nil
	}
	out := *in
	if in.Scopes != nil {
		out.Scopes = append([]string(nil), in.Scopes...)
	}
	if in.Metadata != nil {
		out.Metadata = make(map[string]string, len(in.Metadata))
		for k, v := range in.Metadata {
			out.Metadata[k] = v
		}
	}
	if in.ExpiresAt != nil {
		t := *in.ExpiresAt
		out.ExpiresAt = &t
	}
	if in.RevokedAt != nil {
		t := *in.RevokedAt
		out.RevokedAt = &t
	}
	return &out
}

type CachedRevoker struct {
	base        Revoker
	invalidator LookupInvalidator
}

var _ Revoker = (*CachedRevoker)(nil)

func NewCachedRevoker(base Revoker, invalidator LookupInvalidator) *CachedRevoker {
	return &CachedRevoker{base: base, invalidator: invalidator}
}

func (r *CachedRevoker) RevokeKey(ctx context.Context, keyID string, reason string) error {
	if err := r.base.RevokeKey(ctx, keyID, reason); err != nil {
		return err
	}
	if r.invalidator != nil {
		return r.invalidator.InvalidateKey(ctx, keyID)
	}
	return nil
}

type CachedRotator struct {
	base        Rotator
	invalidator LookupInvalidator
}

var _ Rotator = (*CachedRotator)(nil)

func NewCachedRotator(base Rotator, invalidator LookupInvalidator) *CachedRotator {
	return &CachedRotator{base: base, invalidator: invalidator}
}

func (r *CachedRotator) RotateKey(ctx context.Context, keyID string, prefix string) (*IssuedKey, error) {
	out, err := r.base.RotateKey(ctx, keyID, prefix)
	if err != nil {
		return nil, err
	}
	if r.invalidator != nil {
		if err := r.invalidator.InvalidateKey(ctx, keyID); err != nil {
			return nil, err
		}
		if out != nil && out.KeyID != "" && out.KeyID != keyID {
			if err := r.invalidator.InvalidateKey(ctx, out.KeyID); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
