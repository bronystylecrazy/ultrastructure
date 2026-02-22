package apikey

import (
	"strings"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
)

func UseLookup(lookup KeyLookup) di.Node {
	return di.Supply(lookup, di.As[KeyLookup]())
}

func UseUsageRecorder(recorder KeyUsageRecorder) di.Node {
	return di.Supply(recorder, di.As[KeyUsageRecorder]())
}

func UseRevoker(revoker Revoker) di.Node {
	return di.Supply(revoker, di.As[Revoker]())
}

func UseRotator(rotator Rotator) di.Node {
	return di.Supply(rotator, di.As[Rotator]())
}

func UseHasher(hasher Hasher) di.Node {
	return di.Supply(hasher, di.As[Hasher]())
}

type CacheOption func(*CachedLookupConfig)

func WithCacheL1TTL(ttlSec int) CacheOption {
	return func(c *CachedLookupConfig) {
		if ttlSec > 0 {
			c.L1TTL = time.Duration(ttlSec) * time.Second
		}
	}
}

func WithCacheL2TTL(ttlSec int) CacheOption {
	return func(c *CachedLookupConfig) {
		if ttlSec > 0 {
			c.L2TTL = time.Duration(ttlSec) * time.Second
		}
	}
}

func WithCacheNegativeTTL(ttlSec int) CacheOption {
	return func(c *CachedLookupConfig) {
		if ttlSec > 0 {
			c.NegativeTTL = time.Duration(ttlSec) * time.Second
		}
	}
}

func WithCacheNamespace(namespace string) CacheOption {
	return func(c *CachedLookupConfig) {
		c.Namespace = strings.TrimSpace(namespace)
	}
}

func WithCacheRedisKeyPrefix(prefix string) CacheOption {
	return func(c *CachedLookupConfig) {
		c.RedisKeyPrefix = strings.TrimSpace(prefix)
	}
}

func UseCachedLookup(opts ...CacheOption) di.Node {
	return di.Decorate(func(base KeyLookup, config Config, cacheStore CacheStore) KeyLookup {
		cfg := CachedLookupConfig{
			Namespace: strings.TrimSpace(config.KeyPrefix),
		}
		for _, opt := range opts {
			if opt != nil {
				opt(&cfg)
			}
		}
		return NewCachedLookup(base, cacheStore, cfg)
	}, di.Params(``, ``, di.Optional()))
}

func UseCachedRevoker() di.Node {
	return di.Decorate(func(base Revoker, lookup KeyLookup) Revoker {
		invalidator, _ := lookup.(LookupInvalidator)
		return NewCachedRevoker(base, invalidator)
	})
}

func UseCachedRotator() di.Node {
	return di.Decorate(func(base Rotator, lookup KeyLookup) Rotator {
		invalidator, _ := lookup.(LookupInvalidator)
		return NewCachedRotator(base, invalidator)
	})
}
