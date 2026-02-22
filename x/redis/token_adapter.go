package rd

import (
	"context"
	"errors"
	"time"

	"github.com/bronystylecrazy/ultrastructure/security/session"
	redis "github.com/redis/go-redis/v9"
)

type tokenRevocationCacheAdapter struct {
	client *redis.Client
}

func NewTokenRevocationCache(client *redis.Client) session.RevocationCache {
	if client == nil {
		return nil
	}
	return tokenRevocationCacheAdapter{client: client}
}

func (a tokenRevocationCacheAdapter) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return a.client.Set(ctx, key, value, ttl).Err()
}

func (a tokenRevocationCacheAdapter) Get(ctx context.Context, key string) (string, error) {
	val, err := a.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", session.ErrRevocationCacheMiss
		}
		return "", err
	}
	return val, nil
}
